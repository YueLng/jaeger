// Copyright (c) 2017 Uber Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"github.com/gorilla/handlers"
	"github.com/gorilla/mux"
	basicB "github.com/jaegertracing/jaeger/cmd/builder"
	"github.com/jaegertracing/jaeger/cmd/flags"
	"github.com/jaegertracing/jaeger/cmd/query/app"
	"github.com/jaegertracing/jaeger/cmd/query/app/builder"
	"github.com/jaegertracing/jaeger/pkg/config"
	pMetrics "github.com/jaegertracing/jaeger/pkg/metrics"
	"github.com/jaegertracing/jaeger/pkg/version"
	"github.com/jaegertracing/jaeger/storage/spanstore/memory"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	jaegerClientConfig "github.com/uber/jaeger-client-go/config"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"syscall"
)

func main() {
	var serverChannel = make(chan os.Signal, 0)
	signal.Notify(serverChannel, os.Interrupt, syscall.SIGTERM)

	//casOptions := casFlags.NewOptions("cassandra", "cassandra.archive")
	//esOptions := esFlags.NewOptions("es", "es.archive")
	v := viper.New()

	var command = &cobra.Command{
		Use:   "jaeger-query",
		Short: "Jaeger query is a service to access tracing data",
		Long:  `Jaeger query is a service to access tracing data and host UI.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := flags.TryLoadConfigFile(v)
			if err != nil {
				return err
			}

			sFlags := new(flags.SharedFlags).InitFromViper(v)
			logger := log.New()
			if err != nil {
				return err
			}

			//casOptions.InitFromViper(v)
			//esOptions.InitFromViper(v)
			queryOpts := new(builder.QueryOptions).InitFromViper(v)
			mBldr := new(pMetrics.Builder).InitFromViper(v)

			//hc, err := healthcheck.Serve(http.StatusServiceUnavailable, queryOpts.HealthCheckHTTPPort, logger)
			//if err != nil {
			//	logger.Fatal("Could not start the health check server.", zap.Error(err))
			//}

			metricsFactory, err := mBldr.CreateMetricsFactory("jaeger-query")
			if err != nil {
				logger.Fatal("Cannot create metrics factory.")
			}

			tracer, closer, err := jaegerClientConfig.Configuration{
				Sampler: &jaegerClientConfig.SamplerConfig{
					Type:  "probabilistic",
					Param: 1.0,
				},
				RPCMetrics: true,
			}.New("jaeger-query", jaegerClientConfig.Metrics(metricsFactory))
			if err != nil {
				logger.Fatal("Failed to initialize tracer")
			}
			defer closer.Close()

			storageBuild, err := builder.NewStorageBuilder(
				flags.MemoryStorageType,
				sFlags.DependencyStorage.DataFrequency,
				basicB.Options.MetricsFactoryOption(metricsFactory),
				basicB.Options.MemoryStoreOption(memory.NewStore()),
				//basicB.Options.CassandraSessionOption(casOptions.GetPrimary()),
				//basicB.Options.ElasticClientOption(esOptions.GetPrimary()),
			)
			if err != nil {
				logger.Fatal("Failed to init storage builder")
			}

			apiHandler := app.NewAPIHandler(
				storageBuild.SpanReader,
				storageBuild.DependencyReader,
				storageBuild.SpanWriter,
				app.HandlerOptions.Prefix(queryOpts.Prefix),
				app.HandlerOptions.Tracer(tracer))
			r := mux.NewRouter()
			apiHandler.RegisterRoutes(r)
			registerStaticHandler(r, queryOpts)

			if h := mBldr.Handler(); h != nil {
				logger.Info("Registering metrics handler with HTTP server")
				r.Handle(mBldr.HTTPRoute, h)
			}

			portStr := ":" + strconv.Itoa(queryOpts.Port)
			compressHandler := handlers.CompressHandler(r)
			//recoveryHandler := recoveryhandler.NewRecoveryHandler(logger, true)

			go func() {
				logger.Info("Starting jaeger-query HTTP server")
				if err := http.ListenAndServe(portStr, compressHandler); err != nil {
					logger.Fatal("Could not launch service")
				}
				//hc.Set(http.StatusInternalServerError)
			}()

			//hc.Ready()

			select {
			case <-serverChannel:
				logger.Info("Jaeger Query is finishing")
			}
			return nil
		},
	}

	command.AddCommand(version.Command())

	config.AddFlags(
		v,
		command,
		flags.AddConfigFileFlag,
		flags.AddFlags,
		//casOptions.AddFlags,
		//esOptions.AddFlags,
		pMetrics.AddFlags,
		builder.AddFlags,
	)

	if error := command.Execute(); error != nil {
		fmt.Println(error.Error())
		os.Exit(1)
	}
}

func registerStaticHandler(r *mux.Router, qOpts *builder.QueryOptions) {
	staticHandler, err := app.NewStaticAssetsHandler(qOpts.StaticAssets, qOpts.UIConfig)
	if err != nil {
		log.Fatal("Could not create static assets handler")
	}
	if staticHandler != nil {
		staticHandler.RegisterRoutes(r)
	} else {
		log.Info("Static handler is not registered")
	}
}
