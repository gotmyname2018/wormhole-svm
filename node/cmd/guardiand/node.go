package guardiand

import (
	"context"
	"fmt"
	"net"
	_ "net/http/pprof" // #nosec G108 we are using a custom router (`router := mux.NewRouter()`) and thus not automatically expose pprof.
	"os"
	"os/signal"
	"path"
	"runtime"
	"syscall"
	"time"

	"github.com/certusone/wormhole/node/pkg/watchers"
	ethcrypto "github.com/ethereum/go-ethereum/crypto"

	"github.com/certusone/wormhole/node/pkg/watchers/solana"

	"github.com/certusone/wormhole/node/pkg/db"
	"github.com/certusone/wormhole/node/pkg/telemetry"
	"github.com/certusone/wormhole/node/pkg/version"
	"github.com/gagliardetto/solana-go/rpc"
	"go.uber.org/zap/zapcore"

	"github.com/certusone/wormhole/node/pkg/common"
	"github.com/certusone/wormhole/node/pkg/devnet"
	"github.com/certusone/wormhole/node/pkg/node"
	"github.com/certusone/wormhole/node/pkg/p2p"
	"github.com/certusone/wormhole/node/pkg/supervisor"
	promremotew "github.com/certusone/wormhole/node/pkg/telemetry/prom_remote_write"
	libp2p_crypto "github.com/libp2p/go-libp2p/core/crypto"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/spf13/cobra"
	"github.com/wormhole-foundation/wormhole/sdk/vaa"
	"go.uber.org/zap"

	ipfslog "github.com/ipfs/go-log/v2"
)

var (
	p2pNetworkID *string
	p2pPort      *uint
	p2pBootstrap *string

	nodeKeyPath *string

	adminSocketPath      *string
	publicGRPCSocketPath *string

	dataDir *string

	statusAddr *string

	guardianKeyPath *string
	solanaContract  *string

	solanaRPC *string

	logLevel                *string
	publicRpcLogDetailStr   *string
	publicRpcLogToTelemetry *bool

	unsafeDevMode *bool
	testnetMode   *bool
	nodeName      *string

	publicRPC *string
	publicWeb *string

	tlsHostname *string
	tlsProdEnv  *bool

	disableHeartbeatVerify *bool

	disableTelemetry *bool

	// Loki cloud logging parameters
	telemetryLokiURL *string

	// Prometheus remote write URL
	promRemoteURL *string

	chainGovernorEnabled *bool

	ccqEnabled           *bool
	ccqAllowedRequesters *string
	ccqP2pPort           *uint
	ccqP2pBootstrap      *string
	ccqAllowedPeers      *string
	ccqBackfillCache     *bool
)

func init() {
	p2pNetworkID = NodeCmd.Flags().String("network", "/wormhole/dev", "P2P network identifier")
	p2pPort = NodeCmd.Flags().Uint("port", p2p.DefaultPort, "P2P UDP listener port")
	p2pBootstrap = NodeCmd.Flags().String("bootstrap", "", "P2P bootstrap peers (comma-separated)")

	statusAddr = NodeCmd.Flags().String("statusAddr", "[::]:6060", "Listen address for status server (disabled if blank)")

	nodeKeyPath = NodeCmd.Flags().String("nodeKey", "", "Path to node key (will be generated if it doesn't exist)")

	adminSocketPath = NodeCmd.Flags().String("adminSocket", "", "Admin gRPC service UNIX domain socket path")
	publicGRPCSocketPath = NodeCmd.Flags().String("publicGRPCSocket", "", "Public gRPC service UNIX domain socket path")

	dataDir = NodeCmd.Flags().String("dataDir", "", "Data directory")

	guardianKeyPath = NodeCmd.Flags().String("guardianKey", "", "Path to guardian key (required)")
	solanaContract = NodeCmd.Flags().String("solanaContract", "", "Address of the Solana program (required)")

	solanaRPC = node.RegisterFlagWithValidationOrFail(NodeCmd, "solanaRPC", "Solana RPC URL (required)", "http://solana-devnet:8899", []string{"http", "https"})

	logLevel = NodeCmd.Flags().String("logLevel", "info", "Logging level (debug, info, warn, error, dpanic, panic, fatal)")
	publicRpcLogDetailStr = NodeCmd.Flags().String("publicRpcLogDetail", "full", "The detail with which public RPC requests shall be logged (none=no logging, minimal=only log gRPC methods, full=log gRPC method, payload (up to 200 bytes) and user agent (up to 200 bytes))")
	publicRpcLogToTelemetry = NodeCmd.Flags().Bool("logPublicRpcToTelemetry", true, "whether or not to include publicRpc request logs in telemetry")

	unsafeDevMode = NodeCmd.Flags().Bool("unsafeDevMode", false, "Launch node in unsafe, deterministic devnet mode")
	testnetMode = NodeCmd.Flags().Bool("testnetMode", false, "Launch node in testnet mode (enables testnet-only features)")
	nodeName = NodeCmd.Flags().String("nodeName", "", "Node name to announce in gossip heartbeats")

	publicRPC = NodeCmd.Flags().String("publicRPC", "", "Listen address for public gRPC interface")
	publicWeb = NodeCmd.Flags().String("publicWeb", "", "Listen address for public REST and gRPC Web interface")

	tlsHostname = NodeCmd.Flags().String("tlsHostname", "", "If set, serve publicWeb as TLS with this hostname using Let's Encrypt")
	tlsProdEnv = NodeCmd.Flags().Bool("tlsProdEnv", false,
		"Use the production Let's Encrypt environment instead of staging")

	disableHeartbeatVerify = NodeCmd.Flags().Bool("disableHeartbeatVerify", false,
		"Disable heartbeat signature verification (useful during network startup)")
	disableTelemetry = NodeCmd.Flags().Bool("disableTelemetry", false,
		"Disable telemetry")

	telemetryLokiURL = NodeCmd.Flags().String("telemetryLokiURL", "", "Loki cloud logging URL")

	promRemoteURL = NodeCmd.Flags().String("promRemoteURL", "", "Prometheus remote write URL (Grafana)")

	chainGovernorEnabled = NodeCmd.Flags().Bool("chainGovernorEnabled", false, "Run the chain governor")

	ccqEnabled = NodeCmd.Flags().Bool("ccqEnabled", false, "Enable cross chain query support")
	ccqAllowedRequesters = NodeCmd.Flags().String("ccqAllowedRequesters", "", "Comma separated list of signers allowed to submit cross chain queries")
	ccqP2pPort = NodeCmd.Flags().Uint("ccqP2pPort", 8996, "CCQ P2P UDP listener port")
	ccqP2pBootstrap = NodeCmd.Flags().String("ccqP2pBootstrap", "", "CCQ P2P bootstrap peers (comma-separated)")
	ccqAllowedPeers = NodeCmd.Flags().String("ccqAllowedPeers", "", "CCQ allowed P2P peers (comma-separated)")
	ccqBackfillCache = NodeCmd.Flags().Bool("ccqBackfillCache", true, "Should EVM chains backfill CCQ timestamp cache on startup")
}

var (
	rootCtx       context.Context
	rootCtxCancel context.CancelFunc
)

var (
	configFilename = "guardiand"
	configPath     = "node/config"
	envPrefix      = "GUARDIAND"
)

// "Why would anyone do this?" are famous last words.
//
// We already forcibly override RPC URLs and keys in dev mode to prevent security
// risks from operator error, but an extra warning won't hurt.
const devwarning = `
        +++++++++++++++++++++++++++++++++++++++++++++++++++
        |   NODE IS RUNNING IN INSECURE DEVELOPMENT MODE  |
        |                                                 |
        |      Do not use --unsafeDevMode in prod.        |
        +++++++++++++++++++++++++++++++++++++++++++++++++++

`

// NodeCmd represents the node command
var NodeCmd = &cobra.Command{
	Use:               "node",
	Short:             "Run the guardiand node",
	PersistentPreRunE: initConfig,
	Run:               runNode,
}

// This variable may be overridden by the -X linker flag to "dev" in which case
// we enforce the --unsafeDevMode flag. Only development binaries/docker images
// are distributed. Production binaries are required to be built from source by
// guardians to reduce risk from a compromised builder.
var Build = "prod"

// initConfig initializes the file configuration.
func initConfig(cmd *cobra.Command, args []string) error {
	return node.InitFileConfig(cmd, node.ConfigOptions{
		FilePath:  configPath,
		FileName:  configFilename,
		EnvPrefix: envPrefix,
	})
}

func runNode(cmd *cobra.Command, args []string) {
	if Build == "dev" && !*unsafeDevMode {
		fmt.Println("This is a development build. --unsafeDevMode must be enabled.")
		os.Exit(1)
	}

	if *unsafeDevMode {
		fmt.Print(devwarning)
	}

	if *testnetMode || *unsafeDevMode {
		fmt.Println("Not locking in memory.")
	} else {
		common.LockMemory()
	}

	common.SetRestrictiveUmask()

	// Refuse to run as root in production mode.
	if !*unsafeDevMode && os.Geteuid() == 0 {
		fmt.Println("can't run as uid 0")
		os.Exit(1)
	}

	// Set up logging. The go-log zap wrapper that libp2p uses is compatible with our
	// usage of zap in supervisor, which is nice.
	lvl, err := ipfslog.LevelFromString(*logLevel)
	if err != nil {
		fmt.Println("Invalid log level")
		os.Exit(1)
	}

	logger := zap.New(zapcore.NewCore(
		consoleEncoder{zapcore.NewConsoleEncoder(
			zap.NewDevelopmentEncoderConfig())},
		zapcore.AddSync(zapcore.Lock(os.Stderr)),
		zap.NewAtomicLevelAt(zapcore.Level(lvl))))

	if *unsafeDevMode {
		// Use the hostname as nodeName. For production, we don't want to do this to
		// prevent accidentally leaking sensitive hostnames.
		hostname, err := os.Hostname()
		if err != nil {
			panic(err)
		}
		*nodeName = hostname

		// Put node name into the log for development.
		logger = logger.Named(*nodeName)
	}

	// Override the default go-log config, which uses a magic environment variable.
	ipfslog.SetAllLoggers(lvl)

	// In devnet mode, we automatically set a number of flags that rely on deterministic keys.
	if *unsafeDevMode {
		g0key, err := peer.IDFromPrivateKey(devnet.DeterministicP2PPrivKeyByIndex(0))
		if err != nil {
			panic(err)
		}

		// Use the first guardian node as bootstrap
		*p2pBootstrap = fmt.Sprintf("/dns4/guardian-0.guardian/udp/%d/quic/p2p/%s", *p2pPort, g0key.String())
		*ccqP2pBootstrap = fmt.Sprintf("/dns4/guardian-0.guardian/udp/%d/quic/p2p/%s", *ccqP2pPort, g0key.String())
	}

	// Verify flags

	if *nodeKeyPath == "" && !*unsafeDevMode { // In devnet mode, keys are deterministically generated.
		logger.Fatal("Please specify --nodeKey")
	}
	if *guardianKeyPath == "" {
		logger.Fatal("Please specify --guardianKey")
	}
	if *adminSocketPath == "" {
		logger.Fatal("Please specify --adminSocket")
	}
	if *adminSocketPath == *publicGRPCSocketPath {
		logger.Fatal("--adminSocket must not equal --publicGRPCSocket")
	}
	if (*publicRPC != "" || *publicWeb != "") && *publicGRPCSocketPath == "" {
		logger.Fatal("If either --publicRPC or --publicWeb is specified, --publicGRPCSocket must also be specified")
	}
	if *dataDir == "" {
		logger.Fatal("Please specify --dataDir")
	}

	var publicRpcLogDetail common.GrpcLogDetail
	switch *publicRpcLogDetailStr {
	case "none":
		publicRpcLogDetail = common.GrpcLogDetailNone
	case "minimal":
		publicRpcLogDetail = common.GrpcLogDetailMinimal
	case "full":
		publicRpcLogDetail = common.GrpcLogDetailFull
	default:
		logger.Fatal("--publicRpcLogDetail should be one of (none, minimal, full)")
	}

	if *nodeName == "" {
		logger.Fatal("Please specify --nodeName")
	}

	// Solana, Terra Classic, Terra 2, and Algorand are optional in devnet
	if !*unsafeDevMode {

		if *solanaContract == "" {
			logger.Fatal("Please specify --solanaContract")
		}
		if *solanaRPC == "" {
			logger.Fatal("Please specify --solanaRPC")
		}
	}

	// Determine execution mode
	// TODO: refactor usage of these variables elsewhere. *unsafeDevMode and *testnetMode should not be accessed directly.
	var env common.Environment
	if *unsafeDevMode {
		env = common.UnsafeDevNet
	} else if *testnetMode {
		env = common.TestNet
	} else {
		env = common.MainNet
	}

	if *unsafeDevMode && *testnetMode {
		logger.Fatal("Cannot be in unsafeDevMode and testnetMode at the same time.")
	}

	// In devnet mode, we generate a deterministic guardian key and write it to disk.
	if *unsafeDevMode {
		err := devnet.GenerateAndStoreDevnetGuardianKey(*guardianKeyPath)
		if err != nil {
			logger.Fatal("failed to generate devnet guardian key", zap.Error(err))
		}
	}

	// Database
	db := db.OpenDb(logger, dataDir)
	defer db.Close()

	// Guardian key
	gk, err := common.LoadGuardianKey(*guardianKeyPath, *unsafeDevMode)
	if err != nil {
		logger.Fatal("failed to load guardian key", zap.Error(err))
	}

	logger.Info("Loaded guardian key", zap.String(
		"address", ethcrypto.PubkeyToAddress(gk.PublicKey).String()))

	// Load p2p private key
	var p2pKey libp2p_crypto.PrivKey
	if *unsafeDevMode {
		idx, err := devnet.GetDevnetIndex()
		if err != nil {
			logger.Fatal("Failed to parse hostname - are we running in devnet?")
		}
		p2pKey = devnet.DeterministicP2PPrivKeyByIndex(int64(idx))

		if idx != 0 {
			// try to connect to guardian-0
			for {
				_, err := net.LookupIP("guardian-0.guardian")
				if err == nil {
					break
				}
				logger.Info("Error resolving guardian-0.guardian. Trying again...")
				time.Sleep(time.Second)
			}
			// TODO this is a hack. If this is not the bootstrap Guardian, we wait 10s such that the bootstrap Guardian has enough time to start.
			// This may no longer be necessary because now the p2p.go ensures that it can connect to at least one bootstrap peer and will
			// exit the whole guardian if it is unable to. Sleeping here for a bit may reduce overall startup time by preventing unnecessary restarts, though.
			logger.Info("This is not a bootstrap Guardian. Waiting another 10 seconds for the bootstrap guardian to come online.")
			time.Sleep(time.Second * 10)
		}
	} else {
		p2pKey, err = common.GetOrCreateNodeKey(logger, *nodeKeyPath)
		if err != nil {
			logger.Fatal("Failed to load node key", zap.Error(err))
		}
	}

	rpcMap := make(map[string]string)
	rpcMap["solanaRPC"] = *solanaRPC

	// Node's main lifecycle context.
	rootCtx, rootCtxCancel = context.WithCancel(context.Background())
	defer rootCtxCancel()

	// Handle SIGTERM
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGTERM)
	go func() {
		<-sigterm
		logger.Info("Received sigterm. exiting.")
		rootCtxCancel()
	}()

	usingLoki := *telemetryLokiURL != ""

	var hasTelemetryCredential bool = usingLoki

	// Telemetry is enabled by default in mainnet/testnet. In devnet it is disabled by default
	if !*disableTelemetry && (!*unsafeDevMode || *unsafeDevMode && hasTelemetryCredential) {
		if !hasTelemetryCredential {
			logger.Fatal("Please specify --telemetryLokiURL or set --disableTelemetry=false")
		}

		// Get libp2p peer ID from private key
		pk := p2pKey.GetPublic()
		peerID, err := peer.IDFromPublicKey(pk)
		if err != nil {
			logger.Fatal("Failed to get peer ID from private key", zap.Error(err))
		}

		labels := map[string]string{
			"node_name":     *nodeName,
			"node_key":      peerID.String(),
			"guardian_addr": ethcrypto.PubkeyToAddress(gk.PublicKey).String(),
			"network":       *p2pNetworkID,
			"version":       version.Version(),
		}

		skipPrivateLogs := !*publicRpcLogToTelemetry

		var tm *telemetry.Telemetry
		if usingLoki {
			logger.Info("Using Loki telemetry logger",
				zap.String("publicRpcLogDetail", *publicRpcLogDetailStr),
				zap.Bool("logPublicRpcToTelemetry", *publicRpcLogToTelemetry))

			tm, err = telemetry.NewLokiCloudLogger(context.Background(), logger, *telemetryLokiURL, "wormhole", skipPrivateLogs, labels)
			if err != nil {
				logger.Fatal("Failed to initialize telemetry", zap.Error(err))
			}
		}

		defer tm.Close()
		logger = tm.WrapLogger(logger) // Wrap logger with telemetry logger
	}

	// log golang version
	logger.Info("golang version", zap.String("golang_version", runtime.Version()))

	// Redirect ipfs logs to plain zap
	ipfslog.SetPrimaryCore(logger.Core())

	usingPromRemoteWrite := *promRemoteURL != ""
	if usingPromRemoteWrite {
		var info promremotew.PromTelemetryInfo
		info.PromRemoteURL = *promRemoteURL
		info.Labels = map[string]string{
			"node_name":     *nodeName,
			"guardian_addr": ethcrypto.PubkeyToAddress(gk.PublicKey).String(),
			"network":       *p2pNetworkID,
			"version":       version.Version(),
			"product":       "wormhole",
		}

		promLogger := logger.With(zap.String("component", "prometheus_scraper"))
		errC := make(chan error)
		common.StartRunnable(rootCtx, errC, false, "prometheus_scraper", func(ctx context.Context) error {
			t := time.NewTicker(15 * time.Second)

			for {
				select {
				case <-ctx.Done():
					return nil
				case <-t.C:
					err := promremotew.ScrapeAndSendLocalMetrics(ctx, info, promLogger)
					if err != nil {
						promLogger.Error("ScrapeAndSendLocalMetrics error", zap.Error(err))
						continue
					}
				}
			}
		})
	}

	var watcherConfigs = []watchers.WatcherConfig{}

	if shouldStart(solanaRPC) {
		// confirmed watcher
		wc := &solana.WatcherConfig{
			NetworkID:     "solana-confirmed",
			ChainID:       vaa.ChainIDSolana,
			Rpc:           *solanaRPC,
			Websocket:     "",
			Contract:      *solanaContract,
			ReceiveObsReq: false,
			Commitment:    rpc.CommitmentConfirmed,
		}

		watcherConfigs = append(watcherConfigs, wc)

		// finalized watcher
		wc = &solana.WatcherConfig{
			NetworkID:     "solana-finalized",
			ChainID:       vaa.ChainIDSolana,
			Rpc:           *solanaRPC,
			Websocket:     "",
			Contract:      *solanaContract,
			ReceiveObsReq: true,
			Commitment:    rpc.CommitmentFinalized,
		}
		watcherConfigs = append(watcherConfigs, wc)
	}

	guardianNode := node.NewGuardianNode(
		env,
		gk,
	)

	guardianOptions := []*node.GuardianOption{
		node.GuardianOptionDatabase(db),
		node.GuardianOptionWatchers(watcherConfigs),
		node.GuardianOptionGovernor(*chainGovernorEnabled),
		node.GuardianOptionQueryHandler(*ccqEnabled, *ccqAllowedRequesters),
		node.GuardianOptionAdminService(*adminSocketPath, rpcMap),
		node.GuardianOptionP2P(p2pKey, *p2pNetworkID, *p2pBootstrap, *nodeName, *disableHeartbeatVerify, *p2pPort, *ccqP2pBootstrap, *ccqP2pPort, *ccqAllowedPeers),
		node.GuardianOptionStatusServer(*statusAddr),
		node.GuardianOptionProcessor(),
	}

	if shouldStart(publicGRPCSocketPath) {
		guardianOptions = append(guardianOptions, node.GuardianOptionPublicRpcSocket(*publicGRPCSocketPath, publicRpcLogDetail))

		if shouldStart(publicRPC) {
			guardianOptions = append(guardianOptions, node.GuardianOptionPublicrpcTcpService(*publicRPC, publicRpcLogDetail))
		}

		if shouldStart(publicWeb) {
			guardianOptions = append(guardianOptions,
				node.GuardianOptionPublicWeb(*publicWeb, *publicGRPCSocketPath, *tlsHostname, *tlsProdEnv, path.Join(*dataDir, "autocert")),
			)
		}
	}

	// Run supervisor with Guardian Node as root.
	supervisor.New(rootCtx, logger, guardianNode.Run(rootCtxCancel, guardianOptions...),
		// It's safer to crash and restart the process in case we encounter a panic,
		// rather than attempting to reschedule the runnable.
		supervisor.WithPropagatePanic)

	<-rootCtx.Done()
	logger.Info("root context cancelled, exiting...")
}

func shouldStart(rpc *string) bool {
	return *rpc != "" && *rpc != "none"
}
