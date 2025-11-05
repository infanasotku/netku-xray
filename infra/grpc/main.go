package grpc

import (
	"fmt"
	"os"

	grpc_logrus "github.com/grpc-ecosystem/go-grpc-middleware/logging/logrus"
	grpc_ctxtags "github.com/grpc-ecosystem/go-grpc-middleware/tags"
	"github.com/infanasotku/netku/services/xray/contracts"
	"github.com/infanasotku/netku/services/xray/infra/grpc/gen"
	"github.com/sirupsen/logrus"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/health"
	healthgrpc "google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/reflection"
)

func getCredentials(clientCredentials bool) (credentials.TransportCredentials, error) {
	keyfile := os.Getenv("SSL_KEYFILE")
	certfile := os.Getenv("SSL_CERTFILE")

	if clientCredentials {
		return credentials.NewClientTLSFromFile(certfile, "")
	}
	return credentials.NewServerTLSFromFile(certfile, keyfile)
}

func CreateGPRCServer(logger *logrus.Logger, withReflection bool) (*grpc.Server, error) {
	creds, err := getCredentials(false)
	if err != nil {
		return nil, fmt.Errorf("failed to load credentials: %v", err)
	}

	logrusEntry := logrus.NewEntry(logger)
	opts := []grpc_logrus.Option{
		grpc_logrus.WithLevels(func(code codes.Code) logrus.Level {
			if code == codes.OK {
				return logrus.InfoLevel
			}
			return logrus.ErrorLevel
		}),
	}
	grpc_logrus.ReplaceGrpcLogger(logrusEntry)

	grpcServer := grpc.NewServer(grpc.Creds(creds), grpc.ChainUnaryInterceptor(grpc_ctxtags.UnaryServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
		grpc_logrus.UnaryServerInterceptor(logrusEntry, opts...)),
		grpc.ChainStreamInterceptor(
			grpc_ctxtags.StreamServerInterceptor(grpc_ctxtags.WithFieldExtractor(grpc_ctxtags.CodeGenRequestFieldExtractor)),
			grpc_logrus.StreamServerInterceptor(logrusEntry, opts...),
		))

	if withReflection {
		reflection.Register(grpcServer)
	}

	return grpcServer, nil
}

func BindXrayServer(server *grpc.Server, xrayService contracts.XrayService, logger *logrus.Logger) {
	xray_server := XrayServer{xrayService: xrayService, logger: logger}

	gen.RegisterXrayServer(server, &xray_server)
}

func BindHealthCheck(server *grpc.Server) {
	healthcheck := health.NewServer()
	healthgrpc.RegisterHealthServer(server, healthcheck)
	healthcheck.SetServingStatus("xray", healthgrpc.HealthCheckResponse_SERVING)
}
