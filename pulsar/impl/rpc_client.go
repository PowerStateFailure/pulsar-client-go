package impl

import (
	"github.com/golang/protobuf/proto"
	"net/url"
	pb "pulsar-client-go-native/pulsar/pulsar_proto"
	"sync"
	"sync/atomic"
)

type RpcResult struct {
	Response *pb.BaseCommand
	Cnx      Connection
}

type RpcClient interface {
	// Create a new unique request id
	NewRequestId() uint64

	NewProducerId() uint64

	NewConsumerId() uint64

	// Send a request and block until the result is available
	RequestToAnyBroker(requestId uint64, cmdType pb.BaseCommand_Type, message proto.Message) (*RpcResult, error)

	Request(logicalAddr *url.URL, physicalAddr *url.URL, requestId uint64,
		cmdType pb.BaseCommand_Type, message proto.Message) (*RpcResult, error)

	RequestOnCnx(cnx Connection, requestId uint64, cmdType pb.BaseCommand_Type, message proto.Message) (*RpcResult, error)
}

type rpcClient struct {
	serviceUrl          *url.URL
	pool                ConnectionPool
	requestIdGenerator  uint64
	producerIdGenerator uint64
	consumerIdGenerator uint64
}

func NewRpcClient(serviceUrl *url.URL, pool ConnectionPool) RpcClient {
	return &rpcClient{
		serviceUrl: serviceUrl,
		pool:       pool,
	}
}

func (c *rpcClient) RequestToAnyBroker(requestId uint64, cmdType pb.BaseCommand_Type, message proto.Message) (*RpcResult, error) {
	return c.Request(c.serviceUrl, c.serviceUrl, requestId, cmdType, message)
}

func (c *rpcClient) Request(logicalAddr *url.URL, physicalAddr *url.URL, requestId uint64,
	cmdType pb.BaseCommand_Type, message proto.Message) (*RpcResult, error) {
	// TODO: Add retry logic in case of connection issues
	cnx, err := c.pool.GetConnection(logicalAddr, physicalAddr)
	if err != nil {
		return nil, err
	}

	wg := sync.WaitGroup{}
	wg.Add(1)

	rpcResult := &RpcResult{
		Cnx: cnx,
	}

	// TODO: Handle errors with disconnections
	cnx.SendRequest(requestId, baseCommand(cmdType, message), func(response *pb.BaseCommand) {
		rpcResult.Response = response
		wg.Done()
	})

	wg.Wait()
	return rpcResult, nil
}

func (c *rpcClient) RequestOnCnx(cnx Connection, requestId uint64, cmdType pb.BaseCommand_Type,
	message proto.Message) (*RpcResult, error) {
	wg := sync.WaitGroup{}
	wg.Add(1)

	rpcResult := &RpcResult{
		Cnx: cnx,
	}

	cnx.SendRequest(requestId, baseCommand(cmdType, message), func(response *pb.BaseCommand) {
		rpcResult.Response = response
		wg.Done()
	})

	wg.Wait()
	return rpcResult, nil
}

func (c *rpcClient) NewRequestId() uint64 {
	return atomic.AddUint64(&c.requestIdGenerator, 1)
}

func (c *rpcClient) NewProducerId() uint64 {
	return atomic.AddUint64(&c.producerIdGenerator, 1)
}

func (c *rpcClient) NewConsumerId() uint64 {
	return atomic.AddUint64(&c.consumerIdGenerator, 1)
}
