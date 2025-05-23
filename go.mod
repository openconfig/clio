module github.com/openconfig/clio

go 1.22

require (
	github.com/google/go-cmp v0.7.0
	github.com/open-telemetry/opentelemetry-collector-contrib/receiver/dockerstatsreceiver v0.102.0
	github.com/openconfig/gnmi v0.11.0
	github.com/openconfig/magna v0.0.0-20240326180454-518e16696c84
	github.com/stretchr/testify v1.9.0
	go.opentelemetry.io/collector/component v0.102.1
	go.opentelemetry.io/collector/confmap v0.102.1
	go.opentelemetry.io/collector/confmap/provider/fileprovider v0.100.0
	go.opentelemetry.io/collector/connector v0.100.0
	go.opentelemetry.io/collector/consumer v0.102.1
	go.opentelemetry.io/collector/exporter v0.100.0
	go.opentelemetry.io/collector/extension v0.102.1
	go.opentelemetry.io/collector/otelcol v0.100.0
	go.opentelemetry.io/collector/pdata v1.9.0
	go.opentelemetry.io/collector/processor v0.100.0
	go.opentelemetry.io/collector/receiver v0.102.0
	go.opentelemetry.io/collector/receiver/otlpreceiver v0.100.0
	go.opentelemetry.io/otel/metric v1.27.0
	go.opentelemetry.io/otel/trace v1.27.0
	go.opentelemetry.io/proto/otlp v1.2.0
	go.uber.org/zap v1.27.0
	google.golang.org/grpc v1.64.1
	google.golang.org/grpc/security/advancedtls v1.0.0
	google.golang.org/protobuf v1.34.1
	k8s.io/klog/v2 v2.120.1
)

require (
	bitbucket.org/creachadair/stringset v0.0.14 // indirect
	github.com/Microsoft/go-winio v0.6.1 // indirect
	github.com/beorn7/perks v1.0.1 // indirect
	github.com/cenkalti/backoff/v4 v4.3.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/davecgh/go-spew v1.1.2-0.20180830191138-d8f796af33cc // indirect
	github.com/distribution/reference v0.5.0 // indirect
	github.com/docker/docker v26.1.5+incompatible // indirect
	github.com/docker/go-connections v0.5.0 // indirect
	github.com/docker/go-units v0.5.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/fsnotify/fsnotify v1.7.0 // indirect
	github.com/go-logr/logr v1.4.2 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-ole/go-ole v1.3.0 // indirect
	github.com/go-viper/mapstructure/v2 v2.0.0-alpha.1 // indirect
	github.com/gobwas/glob v0.2.3 // indirect
	github.com/gogo/protobuf v1.3.2 // indirect
	github.com/golang/glog v1.2.2 // indirect
	github.com/golang/groupcache v0.0.0-20210331224755-41bb18bfe9da // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/grpc-ecosystem/grpc-gateway/v2 v2.20.0 // indirect
	github.com/hashicorp/go-version v1.7.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.17.9 // indirect
	github.com/knadh/koanf/maps v0.1.1 // indirect
	github.com/knadh/koanf/providers/confmap v0.1.0 // indirect
	github.com/knadh/koanf/v2 v2.1.1 // indirect
	github.com/kylelemons/godebug v1.1.0 // indirect
	github.com/lufia/plan9stats v0.0.0-20240513124658-fba389f38bae // indirect
	github.com/mitchellh/copystructure v1.2.0 // indirect
	github.com/mitchellh/reflectwalk v1.0.2 // indirect
	github.com/moby/docker-image-spec v1.3.1 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/mostynb/go-grpc-compression v1.2.3 // indirect
	github.com/open-telemetry/opentelemetry-collector-contrib/internal/docker v0.102.0 // indirect
	github.com/open-traffic-generator/snappi/gosnappi v1.4.0 // indirect
	github.com/openconfig/goyang v1.4.5 // indirect
	github.com/openconfig/ygot v0.29.19 // indirect
	github.com/opencontainers/go-digest v1.0.0 // indirect
	github.com/opencontainers/image-spec v1.1.0 // indirect
	github.com/pkg/errors v0.9.1 // indirect
	github.com/pmezard/go-difflib v1.0.1-0.20181226105442-5d4384ee4fb2 // indirect
	github.com/power-devops/perfstat v0.0.0-20240221224432-82ca36839d55 // indirect
	github.com/prometheus/client_golang v1.19.1 // indirect
	github.com/prometheus/client_model v0.6.1 // indirect
	github.com/prometheus/common v0.53.0 // indirect
	github.com/prometheus/procfs v0.15.0 // indirect
	github.com/rs/cors v1.11.0 // indirect
	github.com/shirou/gopsutil/v3 v3.24.4 // indirect
	github.com/shoenig/go-m1cpu v0.1.6 // indirect
	github.com/spf13/cobra v1.8.1 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/tklauser/go-sysconf v0.3.14 // indirect
	github.com/tklauser/numcpus v0.8.0 // indirect
	github.com/vishvananda/netlink v1.2.1-beta.2 // indirect
	github.com/vishvananda/netns v0.0.4 // indirect
	github.com/yusufpapurcu/wmi v1.2.4 // indirect
	go.opencensus.io v0.24.0 // indirect
	go.opentelemetry.io/collector v0.102.1 // indirect
	go.opentelemetry.io/collector/config/configauth v0.102.1 // indirect
	go.opentelemetry.io/collector/config/configcompression v1.9.0 // indirect
	go.opentelemetry.io/collector/config/configgrpc v0.102.1 // indirect
	go.opentelemetry.io/collector/config/confighttp v0.102.0 // indirect
	go.opentelemetry.io/collector/config/confignet v0.102.1 // indirect
	go.opentelemetry.io/collector/config/configopaque v1.9.0 // indirect
	go.opentelemetry.io/collector/config/configretry v0.100.0 // indirect
	go.opentelemetry.io/collector/config/configtelemetry v0.102.1 // indirect
	go.opentelemetry.io/collector/config/configtls v0.102.1 // indirect
	go.opentelemetry.io/collector/config/internal v0.102.1 // indirect
	go.opentelemetry.io/collector/confmap/converter/expandconverter v0.100.0 // indirect
	go.opentelemetry.io/collector/confmap/provider/envprovider v0.100.0 // indirect
	go.opentelemetry.io/collector/confmap/provider/httpprovider v0.100.0 // indirect
	go.opentelemetry.io/collector/confmap/provider/httpsprovider v0.100.0 // indirect
	go.opentelemetry.io/collector/confmap/provider/yamlprovider v0.100.0 // indirect
	go.opentelemetry.io/collector/extension/auth v0.102.1 // indirect
	go.opentelemetry.io/collector/featuregate v1.9.0 // indirect
	go.opentelemetry.io/collector/filter v0.102.0 // indirect
	go.opentelemetry.io/collector/pdata/testdata v0.102.1 // indirect
	go.opentelemetry.io/collector/semconv v0.102.0 // indirect
	go.opentelemetry.io/collector/service v0.100.0 // indirect
	go.opentelemetry.io/contrib/config v0.7.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.52.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.52.0 // indirect
	go.opentelemetry.io/contrib/propagators/b3 v1.26.0 // indirect
	go.opentelemetry.io/otel v1.27.0 // indirect
	go.opentelemetry.io/otel/bridge/opencensus v1.26.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc v1.27.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.27.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace v1.27.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc v1.27.0 // indirect
	go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.27.0 // indirect
	go.opentelemetry.io/otel/exporters/prometheus v0.49.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdoutmetric v1.27.0 // indirect
	go.opentelemetry.io/otel/exporters/stdout/stdouttrace v1.27.0 // indirect
	go.opentelemetry.io/otel/sdk v1.27.0 // indirect
	go.opentelemetry.io/otel/sdk/metric v1.27.0 // indirect
	go.uber.org/multierr v1.11.0 // indirect
	golang.org/x/crypto v0.31.0 // indirect
	golang.org/x/exp v0.0.0-20240506185415-9bf2ced13842 // indirect
	golang.org/x/mod v0.17.0 // indirect
	golang.org/x/net v0.26.0 // indirect
	golang.org/x/sync v0.10.0 // indirect
	golang.org/x/sys v0.28.0 // indirect
	golang.org/x/text v0.21.0 // indirect
	golang.org/x/tools v0.21.1-0.20240508182429-e35e4ccd0d2d // indirect
	gonum.org/v1/gonum v0.15.0 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20240520151616-dc85e6b867a5 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20240604185151-ef581f913117 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)
