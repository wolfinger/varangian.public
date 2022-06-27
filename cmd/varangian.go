package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/grpc-ecosystem/grpc-gateway/runtime"
	acctService "github.com/wolfinger/varangian/acct/service"
	acctStore "github.com/wolfinger/varangian/acct/store"
	instService "github.com/wolfinger/varangian/inst/service"
	instStore "github.com/wolfinger/varangian/inst/store"
	lotService "github.com/wolfinger/varangian/lot/service"
	lotStore "github.com/wolfinger/varangian/lot/store"
	orgService "github.com/wolfinger/varangian/org/service"
	orgStore "github.com/wolfinger/varangian/org/store"
	grpcPkg "github.com/wolfinger/varangian/pkg/grpc"
	portService "github.com/wolfinger/varangian/port/service"
	portStore "github.com/wolfinger/varangian/port/store"
	stratService "github.com/wolfinger/varangian/strat/service"
	stratStore "github.com/wolfinger/varangian/strat/store"
	txnService "github.com/wolfinger/varangian/txn/service"
	txnStore "github.com/wolfinger/varangian/txn/store"
	versionService "github.com/wolfinger/varangian/version/service"
	"google.golang.org/grpc"
)

const (
	internalGRPCEndpoint = "127.0.0.1:8443"
)

// set default port
func port() string {
	port := os.Getenv("PORT")
	if len(port) == 0 {
		port = "5000"
	}
	return ":" + port
}

// set default db connection string
func dbConn() (*pg.Options, error) {
	dbConnStr := os.Getenv("DB_CONN_STR")
	if len(dbConnStr) == 0 {
		dbConnStr = "postgresql://localhost:5432/varangian"
	}

	dbConn, err := pg.ParseURL(dbConnStr)
	if err != nil {
		return nil, err
	}

	return dbConn, nil
}

func runServer(conn *pg.DB) {
	// create stores
	instStore := instStore.NewStore(conn)
	orgStore := orgStore.NewStore(conn)
	acctStore := acctStore.NewStore(conn)
	portStore := portStore.NewStore(conn)
	stratStore := stratStore.NewStore(conn)
	lotStore := lotStore.NewStore(conn)
	txnStore := txnStore.NewStore(conn)

	// create services
	services := []grpcPkg.Service{
		instService.NewService(instStore),
		orgService.NewService(orgStore),
		acctService.NewService(acctStore),
		portService.NewService(portStore),
		stratService.NewService(stratStore),
		lotService.NewService(lotStore),
		txnService.NewService(txnStore, lotStore),
		versionService.NewService(),
	}

	ln, err := net.Listen("tcp", "127.0.0.1:8443")
	if err != nil {
		log.Fatal(err)
	}
	server := grpc.NewServer()
	for _, service := range services {
		service.RegisterServer(server)
	}

	go func() {
		// run grpc server forever
		if err := server.Serve(ln); err != nil {
			log.Fatal(err.Error())
		}
	}()

	localConn, err := grpc.Dial(internalGRPCEndpoint, grpc.WithInsecure())
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	mux := runtime.NewServeMux(
		runtime.WithMarshalerOption(runtime.MIMEWildcard, &runtime.JSONPb{
			OrigName: false,
			// EmitDefaults: true,
		}),
	)
	for _, service := range services {
		if err := service.RegisterHandler(ctx, mux, localConn); err != nil {
			log.Fatal(err)
		}
	}

	if err := http.ListenAndServe(port(), mux); err != nil {
		log.Fatal(err)
	}
}

// database logger -- TODO: move to separate package
type dbLogger struct{}

func (d dbLogger) BeforeQuery(ctx context.Context, q *pg.QueryEvent) (context.Context, error) {
	return ctx, nil
}

func (d dbLogger) AfterQuery(ctx context.Context, q *pg.QueryEvent) error {
	fq, _ := q.FormattedQuery()
	fmt.Println(string(fq))

	return nil
}

func main() {
	// set timezone to UTC
	os.Setenv("TZ", "UTC")

	// connect to the varangian database
	opt, err := dbConn()
	if err != nil {
		log.Fatal(err)
	}
	conn := pg.Connect(opt)
	conn.Exec("SET TIMEZONE TO 'UTC'")
	conn.AddQueryHook(dbLogger{})

	go runServer(conn)

	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
	sig := <-signals
	log.Printf("Got signal %s", sig)
}
