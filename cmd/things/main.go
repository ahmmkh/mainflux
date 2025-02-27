// Copyright (c) Mainflux
// SPDX-License-Identifier: Apache-2.0

// Package main contains things main function to start the things service.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/go-zoo/bone"
	"github.com/jmoiron/sqlx"
	chclient "github.com/mainflux/callhome/pkg/client"
	"github.com/mainflux/mainflux"
	"github.com/mainflux/mainflux/internal"
	authclient "github.com/mainflux/mainflux/internal/clients/grpc/auth"
	jaegerclient "github.com/mainflux/mainflux/internal/clients/jaeger"
	pgclient "github.com/mainflux/mainflux/internal/clients/postgres"
	redisclient "github.com/mainflux/mainflux/internal/clients/redis"
	"github.com/mainflux/mainflux/internal/env"
	"github.com/mainflux/mainflux/internal/postgres"
	"github.com/mainflux/mainflux/internal/server"
	grpcserver "github.com/mainflux/mainflux/internal/server/grpc"
	httpserver "github.com/mainflux/mainflux/internal/server/http"
	mflog "github.com/mainflux/mainflux/logger"
	gpostgres "github.com/mainflux/mainflux/pkg/groups/postgres"
	"github.com/mainflux/mainflux/pkg/uuid"
	"github.com/mainflux/mainflux/things/clients"
	capi "github.com/mainflux/mainflux/things/clients/api"
	thcache "github.com/mainflux/mainflux/things/clients/cache"
	thevents "github.com/mainflux/mainflux/things/clients/events"
	cpostgres "github.com/mainflux/mainflux/things/clients/postgres"
	localusers "github.com/mainflux/mainflux/things/clients/standalone"
	ctracing "github.com/mainflux/mainflux/things/clients/tracing"
	"github.com/mainflux/mainflux/things/groups"
	gapi "github.com/mainflux/mainflux/things/groups/api"
	chevents "github.com/mainflux/mainflux/things/groups/events"
	gtracing "github.com/mainflux/mainflux/things/groups/tracing"
	tpolicies "github.com/mainflux/mainflux/things/policies"
	papi "github.com/mainflux/mainflux/things/policies/api"
	grpcapi "github.com/mainflux/mainflux/things/policies/api/grpc"
	httpapi "github.com/mainflux/mainflux/things/policies/api/http"
	pcache "github.com/mainflux/mainflux/things/policies/cache"
	pevents "github.com/mainflux/mainflux/things/policies/events"
	ppostgres "github.com/mainflux/mainflux/things/policies/postgres"
	ppracing "github.com/mainflux/mainflux/things/policies/tracing"
	thingspg "github.com/mainflux/mainflux/things/postgres"
	upolicies "github.com/mainflux/mainflux/users/policies"
	"go.opentelemetry.io/otel/trace"
	"golang.org/x/sync/errgroup"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	svcName            = "things"
	envPrefixDB        = "MF_THINGS_DB_"
	envPrefixCache     = "MF_THINGS_CACHE_"
	envPrefixHTTP      = "MF_THINGS_HTTP_"
	envPrefixGRPC      = "MF_THINGS_AUTH_GRPC_"
	defDB              = "things"
	defSvcHTTPPort     = "9000"
	defSvcAuthGRPCPort = "7000"
)

type config struct {
	LogLevel         string `env:"MF_THINGS_LOG_LEVEL"           envDefault:"info"`
	StandaloneID     string `env:"MF_THINGS_STANDALONE_ID"       envDefault:""`
	StandaloneToken  string `env:"MF_THINGS_STANDALONE_TOKEN"    envDefault:""`
	JaegerURL        string `env:"MF_JAEGER_URL"                 envDefault:"http://jaeger:14268/api/traces"`
	CacheKeyDuration string `env:"MF_THINGS_CACHE_KEY_DURATION"  envDefault:"10m"`
	SendTelemetry    bool   `env:"MF_SEND_TELEMETRY"             envDefault:"true"`
	InstanceID       string `env:"MF_THINGS_INSTANCE_ID"         envDefault:""`
	ESURL            string `env:"MF_THINGS_ES_URL"              envDefault:"redis://localhost:6379/0"`
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	g, ctx := errgroup.WithContext(ctx)

	// Create new things configuration
	cfg := config{}
	if err := env.Parse(&cfg); err != nil {
		log.Fatalf("failed to load %s configuration : %s", svcName, err)
	}

	logger, err := mflog.New(os.Stdout, cfg.LogLevel)
	if err != nil {
		log.Fatalf("failed to init logger: %s", err)
	}

	var exitCode int
	defer mflog.ExitWithError(&exitCode)

	if cfg.InstanceID == "" {
		if cfg.InstanceID, err = uuid.New().ID(); err != nil {
			logger.Error(fmt.Sprintf("failed to generate instanceID: %s", err))
			exitCode = 1
			return
		}
	}

	// Create new database for things
	dbConfig := pgclient.Config{Name: defDB}
	if err := dbConfig.LoadEnv(envPrefixDB); err != nil {
		logger.Fatal(err.Error())
	}
	db, err := pgclient.SetupWithConfig(envPrefixDB, *thingspg.Migration(), dbConfig)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer db.Close()

	tp, err := jaegerclient.NewProvider(svcName, cfg.JaegerURL, cfg.InstanceID)
	if err != nil {
		logger.Error(fmt.Sprintf("Failed to init Jaeger: %s", err))
		exitCode = 1
		return
	}
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			logger.Error(fmt.Sprintf("Error shutting down tracer provider: %v", err))
		}
	}()
	tracer := tp.Tracer(svcName)

	// Setup new redis cache client
	cacheclient, err := redisclient.Setup(envPrefixCache)
	if err != nil {
		logger.Error(err.Error())
		exitCode = 1
		return
	}
	defer cacheclient.Close()

	var auth upolicies.AuthServiceClient
	switch cfg.StandaloneID != "" && cfg.StandaloneToken != "" {
	case true:
		auth = localusers.NewAuthService(cfg.StandaloneID, cfg.StandaloneToken)
		logger.Info("Using standalone auth service")
	default:
		authServiceClient, authHandler, err := authclient.Setup(svcName)
		if err != nil {
			logger.Error(err.Error())
			exitCode = 1
			return
		}
		defer authHandler.Close()
		auth = authServiceClient
		logger.Info("Successfully connected to auth grpc server " + authHandler.Secure())
	}

	csvc, gsvc, psvc, err := newService(ctx, db, dbConfig, auth, cacheclient, cfg, tracer, logger)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to create %s service: %s", svcName, err))
		exitCode = 1
		return
	}

	httpServerConfig := server.Config{Port: defSvcHTTPPort}
	if err := env.Parse(&httpServerConfig, env.Options{Prefix: envPrefixHTTP}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s HTTP server configuration : %s", svcName, err))
		exitCode = 1
		return
	}
	mux := bone.New()
	hsp := httpserver.New(ctx, cancel, "things-policies", httpServerConfig, httpapi.MakeHandler(csvc, psvc, mux, logger), logger)
	hsc := httpserver.New(ctx, cancel, "things-clients", httpServerConfig, capi.MakeHandler(csvc, mux, logger, cfg.InstanceID), logger)
	hsg := httpserver.New(ctx, cancel, "things-groups", httpServerConfig, gapi.MakeHandler(gsvc, mux, logger), logger)

	grpcServerConfig := server.Config{Port: defSvcAuthGRPCPort}
	if err := env.Parse(&grpcServerConfig, env.Options{Prefix: envPrefixGRPC}); err != nil {
		logger.Error(fmt.Sprintf("failed to load %s gRPC server configuration : %s", svcName, err))
		exitCode = 1
		return
	}
	registerThingsServiceServer := func(srv *grpc.Server) {
		reflection.Register(srv)
		tpolicies.RegisterAuthServiceServer(srv, grpcapi.NewServer(csvc, psvc))
	}
	gs := grpcserver.New(ctx, cancel, svcName, grpcServerConfig, registerThingsServiceServer, logger)

	if cfg.SendTelemetry {
		chc := chclient.New(svcName, mainflux.Version, logger, cancel)
		go chc.CallHome(ctx)
	}

	// Start all servers
	g.Go(func() error {
		return hsp.Start()
	})

	g.Go(func() error {
		return gs.Start()
	})

	g.Go(func() error {
		return server.StopSignalHandler(ctx, cancel, logger, svcName, hsc, hsg, hsp, gs)
	})

	if err := g.Wait(); err != nil {
		logger.Error(fmt.Sprintf("%s service terminated: %s", svcName, err))
	}
}

func newService(ctx context.Context, db *sqlx.DB, dbConfig pgclient.Config, auth upolicies.AuthServiceClient, cacheClient *redis.Client, cfg config, tracer trace.Tracer, logger mflog.Logger) (clients.Service, groups.Service, tpolicies.Service, error) {
	database := postgres.NewDatabase(db, dbConfig, tracer)
	cRepo := cpostgres.NewRepository(database)
	gRepo := gpostgres.New(database)
	pRepo := ppostgres.NewRepository(database)

	idp := uuid.New()

	kDuration, err := time.ParseDuration(cfg.CacheKeyDuration)
	if err != nil {
		logger.Error(fmt.Sprintf("failed to parse cache key duration: %s", err.Error()))
	}

	policyCache := pcache.NewCache(cacheClient, kDuration)
	thingCache := thcache.NewCache(cacheClient, kDuration)

	psvc := tpolicies.NewService(auth, pRepo, policyCache, idp)
	csvc := clients.NewService(auth, psvc, cRepo, gRepo, thingCache, idp)
	gsvc := groups.NewService(auth, psvc, gRepo, idp)

	csvc, err = thevents.NewEventStoreMiddleware(ctx, csvc, cfg.ESURL)
	if err != nil {
		return nil, nil, nil, err
	}

	gsvc, err = chevents.NewEventStoreMiddleware(ctx, gsvc, cfg.ESURL)
	if err != nil {
		return nil, nil, nil, err
	}

	psvc, err = pevents.NewEventStoreMiddleware(ctx, psvc, cfg.ESURL)
	if err != nil {
		return nil, nil, nil, err
	}

	csvc = ctracing.New(csvc, tracer)
	csvc = capi.LoggingMiddleware(csvc, logger)
	counter, latency := internal.MakeMetrics(svcName, "api")
	csvc = capi.MetricsMiddleware(csvc, counter, latency)

	gsvc = gtracing.New(gsvc, tracer)
	gsvc = gapi.LoggingMiddleware(gsvc, logger)
	counter, latency = internal.MakeMetrics(fmt.Sprintf("%s_groups", svcName), "api")
	gsvc = gapi.MetricsMiddleware(gsvc, counter, latency)
	psvc = ppracing.New(psvc, tracer)
	psvc = papi.LoggingMiddleware(psvc, logger)
	counter, latency = internal.MakeMetrics(fmt.Sprintf("%s_policies", svcName), "api")
	psvc = papi.MetricsMiddleware(psvc, counter, latency)

	return csvc, gsvc, psvc, nil
}
