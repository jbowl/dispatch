//https://www.openbrewerydb.org/documentation/01-listbreweries
// https://stackoverflow.com/questions/56082458/grpc-organization-in-microservices
// http://www.inanzzz.com/index.php/post/fczr/creating-a-simple-grpc-client-and-server-application-with-golang
// https://github.com/public-apis/public-apis
// https://stedolan.github.io/jq/tutorial/

// docker build -f ./build/package/Dockerfile -t dispatch ../

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/signal"
	"sync/atomic"
	"time"

	"github.com/gorilla/mux"
	log "github.com/sirupsen/logrus"
	"google.golang.org/grpc"

	pb "github.com/jbowl/apibrewery"
)

type key int

const (
	requestIDKey key = 0
)

var (
	listenAddr  string
	healthy     int32
	gTLS_bypass string
)

type dispatchServer struct {
	//	db        db.DWDB
	//	aiq       aiq.AIQ
	handler http.Handler
	//	verifykey *rsa.PublicKey
	healthy int64

	client pb.BreweryServiceClient
}

// respondWithDetails - writes a Content-Type:application/problem+json header
//                  and a response body with details parameter
func respondWithDetails(w http.ResponseWriter, details pb.ProblemDetails) {
	w.Header().Set("Content-Type", "application/problem+json")
	w.WriteHeader(details.Status)

	json.NewEncoder(w).Encode(&details)
}

func (s *dispatchServer) NewRouter() *mux.Router {
	r := mux.NewRouter().StrictSlash(true)

	r.HandleFunc("/healthz", s.healthz).Methods("GET")
	r.HandleFunc("/breweries", s.breweries).Methods("GET", "OPTIONS")
	r.HandleFunc("/breweries/search", s.search).Methods("GET")

	return r
}

func (s *dispatchServer) healthz(w http.ResponseWriter, r *http.Request) {
	if h := atomic.LoadInt64(&s.healthy); h == 0 {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {

		ret := struct {
			Uptime string `json:"uptime"`
		}{}

		ret.Uptime = fmt.Sprintf("%s", time.Since(time.Unix(0, h)))

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(&ret)
	}
}

func (s *dispatchServer) search(w http.ResponseWriter, r *http.Request) {
	//	keys := r.URL.Query()

	//	city := keys.Get("by_city")

	log.Printf("breweries")

	filter := pb.Filter{By: r.URL.RawQuery}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stream, err := s.client.SearchBreweries(ctx, &filter)
	if err != nil {
		log.Fatalf("%v.ListFeatures(_) = _, %v", s.client, err)
	}
	for {
		brewery, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("%v.ListFeatures(_) = _, %v", s.client, err)
		}
		log.Printf("Brewery ID: %d name: %q website: %q", brewery.GetId(), brewery.GetName(), brewery.GetWebsiteUrl())
	}

	w.WriteHeader(http.StatusOK)
	w.Write([]byte(r.URL.RawQuery))
}

func enableCors(w *http.ResponseWriter, r *http.Request) {
	origin := r.Header.Get("Origin")
	// if !whitelisted return

	(*w).Header().Set("Access-Control-Allow-Origin", origin)

	//	(*w).Header().Set("Access-Control-Allow-Origin", "*")
	//w.Header().Set("Access-Control-Allow-Origin", "http://localhost:8080/login")

}

func (s *dispatchServer) breweries(w http.ResponseWriter, r *http.Request) {
	//	keys := r.URL.Query()

	//	city := keys.Get("by_city")

	if (*r).Method == "OPTIONS" {
		enableCors(&w, r)
		return
	}

	log.Printf("breweries")
	log.Printf(r.URL.RawQuery)

	filter := pb.Filter{By: r.URL.RawQuery}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	stream, err := s.client.ListBreweries(ctx, &filter)
	if err != nil {
		respondWithDetails(w, pb.ProblemDetails{
			Detail:   err.Error(),
			Type:     "",
			Title:    http.StatusText(http.StatusInternalServerError),
			Status:   http.StatusInternalServerError,
			Instance: "",
		})
		//respondToClientWithError(w, err)
		return
	}

	br := make([]pb.BreweryResult, 0)

	for {
		brewery, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("%v.ListFeatures(_) = _, %v", s.client, err)
		}
		br = append(br, pb.BreweryResult{
			ID:              brewery.GetId(),
			Name:            brewery.GetName(),
			BreweryType:     brewery.GetType(),
			Street:          brewery.GetStreet(),
			City:            brewery.GetCity(),
			State:           brewery.GetState(),
			CountryProvince: brewery.GetCountryprov(),
			PostalCode:      brewery.GetPostalcode(),
			Country:         brewery.GetCountry(),
			Longitude:       brewery.GetLongitude(),
			Latitude:        brewery.GetLatitude(),
			Phone:           brewery.GetPhone(),
			Website:         brewery.GetWebsiteUrl(),
		})

		//log.Printf("Brewery ID: %d name: %q website: %q", brewery.GetId(), brewery.GetName(), brewery.GetWebsiteUrl())
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(&br)

}

func init() {
	// log as JSON not default ASCII
	log.SetFormatter(&log.JSONFormatter{})
	log.SetOutput(os.Stdout)
	// only log warning severity or above
	//	log.SetLevel(log.WarnLevel)

	log.Printf("init")
}

func run() error {
	// runtime environment variables
	listenPort := os.Getenv("PORT")
	gTLS_bypass = os.Getenv("TLS_BYPASS")

	network := os.Getenv("NETWORK")

	if len(listenPort) == 0 || len(gTLS_bypass) == 0 || len(network) == 0 {
		log.WithFields(log.Fields{
			"PORT":       listenPort,
			"TLS_BYPASS": gTLS_bypass,
			"NETWORK":    network,
		}).Fatal("env arg missing")
	}

	log.WithFields(log.Fields{
		"PORT":       listenPort,
		"TLS_BYPASS": gTLS_bypass,
		"NETWORK":    network,
	}).Info("starting up with these settings")

	dispatch := dispatchServer{}

	dispatch.handler = dispatch.NewRouter()

	//router := http.NewServeMux()
	//	router.Handle("/", index())
	//	router.Handle("/healthz", healthz())

	nextRequestID := func() string {
		return fmt.Sprintf("%d", time.Now().UnixNano())
	}

	// for healthz call
	atomic.StoreInt64(&dispatch.healthy, time.Now().UnixNano())

	httpServer := http.Server{
		Addr:         ":" + listenPort,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
		Handler:      tracing(nextRequestID)(logging()(dispatch.NewRouter())),
	}

	///////// grpc ->

	var opts []grpc.DialOption
	opts = append(opts, grpc.WithInsecure())

	//	opts = append(opts, grpc.WithBlock())

	log.Printf("dialing")

	//	conn, err := grpc.Dial("localhost:50051", opts...)
	//conn, err := grpc.Dial("breweryctr:50051", opts...)

	log.Printf(network)

	log.Printf(network + ":50051")

	conn, err := grpc.Dial(network+":50051", opts...)
	if err != nil {
		log.Printf("fail to dial: %v", err)
		//	log.Fatalf("fail to dial: %v", err)
	}
	log.Printf("dialed")
	defer conn.Close()
	dispatch.client = pb.NewBreweryServiceClient(conn)

	///////// <- grpc

	done := make(chan bool)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)

	go func() {
		<-quit
		log.Println("Server is shutting down...")
		atomic.StoreInt32(&healthy, 0)

		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		httpServer.SetKeepAlivesEnabled(false)
		if err := httpServer.Shutdown(ctx); err != nil {
			log.Fatalf("Could not gracefully shutdown the server: %v\n", err)
		}
		close(done)
	}()

	log.Println("Server is ready to handle requests at", listenAddr)
	atomic.StoreInt32(&healthy, 1)
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Could not listen on %s: %v\n", listenAddr, err)
	}

	<-done
	log.Println("Server stopped")

	return nil
}

func main() {
	// not of value as a docker container
	pid := os.Getpid()
	fmt.Printf("pid for %s = %d\n", os.Args[0], pid)

	if err := run(); err != nil {
		fmt.Fprintf(os.Stderr, "%s\n", err)
		os.Exit(1)
	}
}

func index() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintln(w, "Hello, World!")
	})
}

func healthz() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.LoadInt32(&healthy) == 1 {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusServiceUnavailable)
	})
}

func logging() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			//setResponseHeaders(&w, r)
			// needs to be removed and addressed via a whitelist lookup
			enableCors(&w, r)

			defer func() {
				requestID, ok := r.Context().Value(requestIDKey).(string)
				if !ok {
					requestID = "unknown"
				}
				log.Println(requestID, r.Method, r.URL.Path, r.RemoteAddr, r.UserAgent())
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func tracing(nextRequestID func() string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestID := r.Header.Get("X-Request-Id")
			if requestID == "" {
				requestID = nextRequestID()
			}
			ctx := context.WithValue(r.Context(), requestIDKey, requestID)
			w.Header().Set("X-Request-Id", requestID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
