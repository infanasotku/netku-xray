package grpc

import (
	"context"
	"regexp"

	"github.com/infanasotku/netku/services/xray/infra/grpc/gen"
	"github.com/infanasotku/netku/services/xray/services"
	"github.com/sirupsen/logrus"
)

type XrayServer struct {
	gen.UnimplementedXrayServer
	xrayService services.XrayService
	logger      *logrus.Logger
}

func IsValidUUID(uuid string) bool {
	r := regexp.MustCompile("^[a-fA-F0-9]{8}-[a-fA-F0-9]{4}-4[a-fA-F0-9]{3}-[8|9|aA|bB][a-fA-F0-9]{3}-[a-fA-F0-9]{12}$")
	return r.MatchString(uuid)
}

func (s *XrayServer) RestartXray(context context.Context, req *gen.XrayInfo) (*gen.XrayInfo, error) {
	err := s.xrayService.RestartEngine(context, req.Uuid)
	return &gen.XrayInfo{Uuid: req.Uuid}, err
}
