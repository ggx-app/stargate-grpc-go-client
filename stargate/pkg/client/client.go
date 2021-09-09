package client

import (
	"context"
	"errors"
	"fmt"
	"time"

	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/stargate/stargate-grpc-go-client/stargate/pkg/auth"
	pb "github.com/stargate/stargate-grpc-go-client/stargate/pkg/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	"google.golang.org/protobuf/types/known/anypb"
)

type StargateClient struct {
	client       pb.StargateClient
	conn         grpc.ClientConnInterface
	authProvider auth.AuthProvider
	token        string
}

func NewStargateClientWithConn(conn grpc.ClientConnInterface, authProvider auth.AuthProvider) (*StargateClient, error) {
	client := pb.NewStargateClient(conn)

	return &StargateClient{
		client:       client,
		conn:         conn,
		authProvider: authProvider,
	}, nil
}

type Batch struct{}

func (s *StargateClient) ExecuteQuery(query *pb.Query) (*pb.Response, error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second+10)
	defer cancel()
	return s.ExecuteQueryWithContext(query, ctx)
}

func (s *StargateClient) ExecuteQueryWithContext(query *pb.Query, ctx context.Context) (*pb.Response, error) {
	ctx, err := s.setMetadata(ctx)
	if err != nil {
		return nil, err
	}

	resp, err := s.client.ExecuteQuery(ctx, query)
	if err != nil {
		if e, ok := status.FromError(err); ok {
			if e.Code() == codes.Unauthenticated {
				s.token = "" // clear out bad token before getting a new one
				ctx, err := s.setMetadata(ctx)
				if err != nil {
					return nil, err
				}

				resp, err = s.client.ExecuteQuery(ctx, query)
				if err != nil {
					log.WithError(err).Error("Failed to execute query")
					return nil, fmt.Errorf("failed to execute query: %v", err)
				}
			} else {
				log.WithError(err).Error("Failed to execute query")
				return nil, fmt.Errorf("failed to execute query: %v", err)
			}
		}
	}

	return resp, nil
}

func (s *StargateClient) setMetadata(ctx context.Context) (context.Context, error) {
	if s.token == "" {
		token, err := s.authProvider.GetToken(ctx)
		if err != nil {
			log.WithError(err).Error("Failed to get auth token")
			return nil, fmt.Errorf("failed to get auth token: %v", err)
		}
		s.token = token
	}

	md := metadata.New(map[string]string{"x-cassandra-token": s.token})
	ctx = metadata.NewOutgoingContext(ctx, md)
	return ctx, nil
}

func (s *StargateClient) ExecuteBatch(batch *Batch) (*pb.Response, error) {
	return nil, errors.New("not yet implemented")
}

func (s *StargateClient) ExecuteBatchWithContext(batch *Batch, ctx context.Context) (*pb.Response, error) {
	return nil, errors.New("not yet implemented")
}

func ToResultSet(resp *pb.Response) (*pb.ResultSet, error) {
	if resp.GetResultSet() == nil {
		return nil, errors.New("no result set")
	}

	data := resp.GetResultSet().Data
	var resultSet pb.ResultSet
	if err := anypb.UnmarshalTo(data, &resultSet, proto.UnmarshalOptions{}); err != nil {
		log.WithError(err).Error("Could not unmarshal result")
		return nil, fmt.Errorf("could not unmarshal result: %v", err)
	}
	return &resultSet, nil
}