package config

import "github.com/ethersphere/beekeeper/pkg/k8s"

type BeeProfile struct {
	Profile `yaml:",inline"`
	Bee     `yaml:",inline"`
}

type Bee struct {
	APIAddr              *string `yaml:"api-addr"`
	Bootnodes            *string `yaml:"bootnodes"`
	ClefSignerEnable     *bool   `yaml:"clef-signer-enable"`
	ClefSignerEndpoint   *string `yaml:"clef-signer-endpoint"`
	CORSAllowedOrigins   *string `yaml:"cors-allowed-origins"`
	DataDir              *string `yaml:"data-dir"`
	DBCapacity           *uint64 `yaml:"db-capacity"`
	DebugAPIAddr         *string `yaml:"debug-api-addr"`
	DebugAPIEnable       *bool   `yaml:"debug-api-enable:"`
	GatewayMode          *bool   `yaml:"gateway-mode"`
	GlobalPinningEnabled *bool   `yaml:"global-pinning-enabled"`
	NATAddr              *string `yaml:"nat-addr"`
	NetworkID            *uint64 `yaml:"network-id"`
	P2PAddr              *string `yaml:"p2p-addr"`
	P2PQUICEnable        *bool   `yaml:"p2p-quic-enable"`
	P2PWSEnable          *bool   `yaml:"pwp-ws-enable"`
	Password             *string `yaml:"password"`
	PaymentEarly         *uint64 `yaml:"payment-early"`
	PaymentThreshold     *uint64 `yaml:"payment-threshold"`
	PaymentTolerance     *uint64 `yaml:"payment-tolerance"`
	PostageStampAddress  *string `yaml:"postage-stamp-address"`
	PriceOracleAddress   *string `yaml:"price-oracle-address"`
	ResolverOptions      *string `yaml:"resolver-options"`
	Standalone           *bool   `yaml:"standalone"`
	SwapEnable           *bool   `yaml:"swap-enable"`
	SwapEndpoint         *string `yaml:"swap-endpoint"`
	SwapFactoryAddress   *string `yaml:"swap-factory-address"`
	SwapInitialDeposit   *uint64 `yaml:"swap-initial-deposit"`
	TracingEnabled       *bool   `yaml:"tracing-enabled"`
	TracingEndpoint      *string `yaml:"tracing-endpoint"`
	TracingServiceName   *string `yaml:"tracing-service-name"`
	Verbosity            *uint64 `yaml:"verbosity"`
	WelcomeMessage       *string `yaml:"welcome-message"`
}

// TODO: with reflex
func (b *Bee) Export() k8s.Config {

	return k8s.Config{}
}