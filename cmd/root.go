package cmd

import (
	"fmt"
	"git.xx.network/elixxir/incentives-bot/incentives"
	"git.xx.network/elixxir/incentives-bot/storage"
	"github.com/golang/protobuf/proto"
	"github.com/skip2/go-qrcode"
	"github.com/spf13/cobra"
	jww "github.com/spf13/jwalterweatherman"
	"github.com/spf13/viper"
	"gitlab.com/elixxir/client/api"
	"gitlab.com/elixxir/client/interfaces/message"
	"gitlab.com/elixxir/client/interfaces/params"
	"gitlab.com/elixxir/crypto/contact"
	"gitlab.com/elixxir/primitives/fact"
	"gitlab.com/xx_network/primitives/id"
	"gitlab.com/xx_network/primitives/utils"
	"io/ioutil"
	"net"
	"os"
	"time"
)

var (
	cfgFile, logPath string
)

// RootCmd represents the base command when called without any sub-commands
var rootCmd = &cobra.Command{
	Use:   "",
	Short: "",
	Long:  "",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		// Initialize config & logging
		initConfig()
		initLog()

		// Get database parameters
		rawAddr := viper.GetString("dbAddress")
		var addr, port string
		var err error
		if rawAddr != "" {
			addr, port, err = net.SplitHostPort(rawAddr)
			if err != nil {
				jww.FATAL.Panicf("Unable to get database port from %s: %+v", rawAddr, err)
			}
		}

		udRawAddr := viper.GetString("udbDbAddress")
		var udAddr, udPort string
		if rawAddr != "" {
			addr, port, err = net.SplitHostPort(udRawAddr)
			if err != nil {
				jww.FATAL.Panicf("Unable to get database port from %s: %+v", rawAddr, err)
			}
		}

		sp := storage.Params{
			Username: viper.GetString("dbUsername"),
			Password: viper.GetString("dbPassword"),
			DBName:   viper.GetString("dbName"),
			Address:  addr,
			Port:     port,
		}
		udbParams := storage.Params{
			Username: viper.GetString("UdDbUsername"),
			Password: viper.GetString("UdDbPassword"),
			DBName:   viper.GetString("UdDbName"),
			Address:  udAddr,
			Port:     udPort,
		}

		// Initialize storage object
		s, err := storage.NewStorage(sp, udbParams)
		if err != nil {
			jww.FATAL.Panicf("Failed to initialize storage interface: %+v", err)
		}

		// Get session parameters
		sessionPath := viper.GetString("sessionPath")

		// Only require proto user path if session does not exist
		var protoUserJson []byte
		protoUserPath, err := utils.ExpandPath(viper.GetString("protoUserPath"))
		if err != nil {
			jww.FATAL.Fatalf("Failed to read proto path: %+v", err)
		} else if protoUserPath == "" {
			jww.WARN.Printf("protoUserPath is blank - a new session will be generated")
		}

		sessionPass := viper.GetString("sessionPass")
		networkFollowerTimeout := time.Duration(viper.GetInt("networkFollowerTimeout")) * time.Second

		ndfPath := viper.GetString("ndf")
		ndfJSON, err := ioutil.ReadFile(ndfPath)
		if err != nil {
			jww.FATAL.Panicf("Failed to read NDF: %+v", err)
		}

		nwParams := params.GetDefaultNetwork()

		useProto := protoUserPath != "" && utils.Exists(protoUserPath)
		useSession := sessionPath != "" && utils.Exists(sessionPath)

		if !useProto && !useSession {
			err = api.NewClient(string(ndfJSON), sessionPath, []byte(sessionPass), "")
			if err != nil {
				jww.FATAL.Panicf("Failed to create new client: %+v", err)
			}
			useSession = true
		}

		var cl *api.Client
		if useSession {
			//  If the session exists, load & login
			// Create client object
			cl, err = api.Login(sessionPath, []byte(sessionPass), nwParams)
			if err != nil {
				jww.FATAL.Panicf("Failed to initialize client: %+v", err)
			}
		} else if useProto {
			protoUserJson, err = utils.ReadFile(protoUserPath)
			if err != nil {
				jww.FATAL.Fatalf("Failed to read proto user at %s: %+v", protoUserPath, err)
			}

			// If the session does not exist but we have a proto file
			// Log in using the protofile (attempt to rebuild session)
			cl, err = api.LoginWithProtoClient(sessionPath,
				[]byte(sessionPass), protoUserJson, string(ndfJSON), nwParams)
			if err != nil {
				jww.FATAL.Fatalf("Failed to create client: %+v", err)
			}
		} else {
			jww.FATAL.Panicf("Cannot run with no session or proto info")
		}

		// Generate QR code
		qrSize := viper.GetInt("qrSize")
		qrLevel := qrcode.RecoveryLevel(viper.GetInt("qrLevel"))
		qrPath := viper.GetString("qrPath")
		me := cl.GetUser().GetContact()
		username, err := fact.NewFact(fact.Username, "USERNAME-HERE")
		if err != nil {
			jww.FATAL.Panicf("Failed to create username: %+v", err)
		}
		me.Facts = append(me.Facts, username)
		qr, err := me.MakeQR(qrSize, qrLevel)
		if err != nil {
			jww.FATAL.Panicf("Failed to generate QR code: %+v", err)
		}
		// Save the QR code PNG to file
		err = utils.WriteFile(qrPath, qr, utils.FilePerms, utils.DirPerms)
		if err != nil {
			jww.FATAL.Panicf("Failed to write QR code: %+v", err)
		}

		// Create & register callback to confirm any authenticated channel requests
		rcb := func(requestor contact.Contact) {
			rid, err := cl.ConfirmAuthenticatedChannel(requestor)
			if err != nil {
				jww.ERROR.Printf("Failed to confirm authenticated channel to %+v: %+v", requestor, err)
			}
			jww.DEBUG.Printf("Authenticated channel to %+v created over round %d", requestor, rid)

			time.Sleep(100 * time.Millisecond)

			intro := "Thank you for using the xx network incentives bot!  Please send me your code."
			payload := &incentives.CMIXText{
				Version: 0,
				Text:    intro,
			}
			marshalled, err := proto.Marshal(payload)
			if err != nil {
				jww.ERROR.Printf("Failed to marshal payload: %+v", err)
				return
			}

			contact, err := cl.GetAuthenticatedChannelRequest(requestor.ID)
			if err != nil {
				jww.ERROR.Printf("Could not get authenticated channel request info: %+v", err)
				return
			}

			// Create response message
			resp := message.Send{
				Recipient:   contact.ID,
				Payload:     marshalled,
				MessageType: message.XxMessage,
			}

			rids, mid, t, err := cl.SendE2E(resp, params.GetDefaultE2E())
			if err != nil {
				jww.ERROR.Printf("Failed to send message: %+v", err)
				return
			}
			jww.INFO.Printf("Sent intro [%+v] to %+v on rounds %+v [%+v]", mid, requestor, rids, t)
		}
		cl.GetAuthRegistrar().AddGeneralRequestCallback(rcb)

		// Create coupons impl & register listener on zero user for text messages
		impl := incentives.New(s, cl)
		cl.GetSwitchboard().RegisterListener(&id.ZeroUser, message.XxMessage, impl)

		// Start network follower
		err = cl.StartNetworkFollower(networkFollowerTimeout)
		if err != nil {
			jww.FATAL.Panicf("Failed to start network follower: %+v", err)
		}

		// Wait 5ever
		select {}
	},
}

// Execute calls the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		jww.ERROR.Println(err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Flags().StringVarP(&cfgFile, "config", "c", "",
		"Path to load the configuration file from. If not set, this "+
			"file must be named config.yaml and must be located in "+
			"~/.xxnetwork/, /opt/xxnetwork, or /etc/xxnetwork.")
}

// initConfig reads in config file and ENV variables if set.
func initConfig() {
	var err error
	if cfgFile == "" {
		cfgFile, err = utils.SearchDefaultLocations("config.yaml", "xxnetwork")
		if err != nil {
			jww.FATAL.Panicf("Failed to find config file: %+v", err)
		}
	} else {
		cfgFile, err = utils.ExpandPath(cfgFile)
		if err != nil {
			jww.FATAL.Panicf("Failed to expand config file path: %+v", err)
		}
	}
	viper.SetConfigFile(cfgFile)
	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err != nil {
		fmt.Printf("Unable to read config file (%s): %+v", cfgFile, err.Error())
	}
}

// initLog initializes logging thresholds and the log path.
func initLog() {
	vipLogLevel := viper.GetUint("logLevel")

	// Check the level of logs to display
	if vipLogLevel > 1 {
		// Set the GRPC log level
		err := os.Setenv("GRPC_GO_LOG_SEVERITY_LEVEL", "info")
		if err != nil {
			jww.ERROR.Printf("Could not set GRPC_GO_LOG_SEVERITY_LEVEL: %+v", err)
		}

		err = os.Setenv("GRPC_GO_LOG_VERBOSITY_LEVEL", "99")
		if err != nil {
			jww.ERROR.Printf("Could not set GRPC_GO_LOG_VERBOSITY_LEVEL: %+v", err)
		}
		// Turn on trace logs
		jww.SetLogThreshold(jww.LevelTrace)
		jww.SetStdoutThreshold(jww.LevelTrace)
	} else if vipLogLevel == 1 {
		// Turn on debugging logs
		jww.SetLogThreshold(jww.LevelDebug)
		jww.SetStdoutThreshold(jww.LevelDebug)
	} else {
		// Turn on info logs
		jww.SetLogThreshold(jww.LevelInfo)
		jww.SetStdoutThreshold(jww.LevelInfo)
	}

	logPath = viper.GetString("log")

	logFile, err := os.OpenFile(logPath,
		os.O_CREATE|os.O_WRONLY|os.O_APPEND,
		0644)
	if err != nil {
		fmt.Printf("Could not open log file %s!\n", logPath)
	} else {
		jww.SetLogOutput(logFile)
	}
}
