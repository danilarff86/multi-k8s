package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"

	"github.com/go-redis/redis/v8"
	"github.com/gorilla/mux"
	_ "github.com/lib/pq"
)

const (
	REDIS_HOST = "REDIS_HOST"
	REDIS_PORT = "REDIS_PORT"
	PGUSER     = "PGUSER"
	PGHOST     = "PGHOST"
	PGPORT     = "PGPORT"
	PGDATABASE = "PGDATABASE"
	PGPASSWORD = "PGPASSWORD"
)

type value struct {
	Number int `json:"number"`
}

type httpService struct {
	db          *sql.DB
	redisClient redis.UniversalClient
}

func checkErr(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	log.Println("starting Golang server")

	ctx := context.Background()

	db, err := connectToDB(ctx)
	checkErr(err)
	defer db.Close()

	redisClient, err := connectToRedis(ctx)
	checkErr(err)
	defer redisClient.Close()

	service := &httpService{
		db:          db,
		redisClient: redisClient,
	}

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS values (number INT)")
	checkErr(err)

	router := mux.NewRouter()
	router.HandleFunc("/", service.mainHandler).Methods(http.MethodGet)
	router.HandleFunc("/values/all", service.allHandler).Methods(http.MethodGet)
	router.HandleFunc("/values/current", service.currentHandler).Methods(http.MethodGet)
	router.HandleFunc("/values", service.setHandler).Methods(http.MethodPost)

	log.Fatal(http.ListenAndServe(":5000", router))
}

func (s *httpService) mainHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("main handler")

	w.Write([]byte("Hi"))
}

func (s *httpService) allHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("all handler")

	rows, err := s.db.QueryContext(r.Context(), "SELECT * from values")
	if err != nil {
		http.Error(w, fmt.Sprintf("unable to retrieve values from db: %v", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var values []value

	for rows.Next() {
		var val int
		err := rows.Scan(&val)
		if err != nil {
			http.Error(w, fmt.Sprintf("unable to parse values: %v", err), http.StatusInternalServerError)
			return
		}

		values = append(values, value{Number: val})
	}

	err = json.NewEncoder(w).Encode(values)
	if err != nil {
		http.Error(w, fmt.Sprintf("unable to encode values: %v", err), http.StatusInternalServerError)
	}
}

func (s *httpService) currentHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("current handler")

	resCmd := s.redisClient.HGetAll(r.Context(), "values")
	if err := resCmd.Err(); err != nil {
		http.Error(w, fmt.Sprintf("unable to retrieve values redis: %v", err), http.StatusInternalServerError)
		return
	}

	err := json.NewEncoder(w).Encode(resCmd.Val())
	if err != nil {
		http.Error(w, fmt.Sprintf("unable to encode values: %v", err), http.StatusInternalServerError)
	}
}

func (s *httpService) setHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("set handler")

	var indexBody struct {
		Val string `json:"index"`
	}

	err := json.NewDecoder(r.Body).Decode(&indexBody)
	if err != nil {
		http.Error(w, "unable to parse body", http.StatusInternalServerError)
		return
	}
	defer r.Body.Close()

	index, err := strconv.Atoi(string(indexBody.Val))
	if err != nil {
		http.Error(w, fmt.Sprintf("unable to parse index value: %s", indexBody.Val), http.StatusBadRequest)
		return
	}

	if index > 40 {
		http.Error(w, "index too high", http.StatusBadRequest)
		return
	}

	const nothingYet = "Nothing yet!"
	err = s.redisClient.HSet(r.Context(), "values", indexBody.Val, nothingYet).Err()
	if err != nil {
		errMsg := fmt.Sprintf("unable to set value '%s' for key: '%s'", nothingYet, indexBody.Val)
		http.Error(w, errMsg, http.StatusInternalServerError)
		return
	}

	err = s.redisClient.Publish(r.Context(), "insert", index).Err()
	if err != nil {
		http.Error(w, "unable to publish message", http.StatusInternalServerError)
		return
	}

	_, err = s.db.ExecContext(r.Context(), "INSERT INTO values(number) VALUES ($1)", index)
	if err != nil {
		http.Error(w, fmt.Sprintf("unable to save data to DB: %v", err), http.StatusInternalServerError)
		return
	}

	_, err = w.Write([]byte(`"working": true`))
	if err != nil {
		http.Error(w, "unable to write reponse body", http.StatusInternalServerError)
	}
}

func getEnv(name string) string {
	val := os.Getenv(name)
	if val == "" {
		panic(fmt.Sprintf("environment variable '%s' is not set", name))
	}
	return val
}

func connectToDB(ctx context.Context) (*sql.DB, error) {
	dbinfo := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		getEnv(PGHOST), getEnv(PGPORT), getEnv(PGUSER), getEnv(PGPASSWORD), getEnv(PGDATABASE))
	db, err := sql.Open("postgres", dbinfo)
	if err != nil {
		return nil, err
	}

	return db, db.PingContext(ctx)
}

func connectToRedis(ctx context.Context) (redis.UniversalClient, error) {
	client := redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", getEnv(REDIS_HOST), getEnv(REDIS_PORT)),
	})
	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}
	return client, nil
}
