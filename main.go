package main

import (
	"log"
	"net/http"
	"os"
	"time"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"

	redistore "gopkg.in/boj/redistore.v1"

	"github.com/gorilla/mux"
)

// User struct holds a user data
type User struct {
	Username string `bson:"username"`
	Password string `bson:"password"`
}

var (
	rediStore *redistore.RediStore
	mgoConn   *mgo.Session
)

func login(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		http.Error(w, "Please pass the data as URL form encoded", http.StatusBadRequest)
		return
	}
	username := r.PostForm.Get("username")
	password := r.PostForm.Get("password")

	c := mgoConn.DB("appdb").C("users")

	user := User{}

	err = c.Find(bson.M{"username": username, "password": password}).One(&user)
	if err != nil {
		http.Error(w, "Invalid Credentials", http.StatusUnauthorized)
		return
	}
	session, _ := rediStore.Get(r, "session.id")
	session.Values["authenticated"] = true
	session.Save(r, w)
	w.Write([]byte("Logged In successfully!"))
}

func logout(w http.ResponseWriter, r *http.Request) {
	session, _ := rediStore.Get(r, "session.id")
	session.Options.MaxAge = -1
	session.Save(r, w)
	w.Write([]byte("Logged Out successfully"))
}

func healthCheck(w http.ResponseWriter, r *http.Request) {
	session, _ := rediStore.Get(r, "session.id")
	if session.Values["authenticated"] == nil {
		http.Error(w, "403 - Forbidden", http.StatusForbidden)
		return
	}
	w.Write([]byte(time.Now().String()))
}

func init() {
	var err error
	rediStore, err = redistore.NewRediStore(10, "tcp", ":6379", "",
		[]byte(os.Getenv("SESSION_SECRET")))
	if err != nil {
		log.Printf("Error connecting to redis: %s\n", err)
	}
	rediStore.SetMaxAge(10 * 24 * 3600)

	mgoConn, err = mgo.Dial("127.0.0.1")
	if err != nil {
		log.Printf("Error connecting to redis: %s\n", err)
	}
}

func main() {
	defer rediStore.Close()
	defer mgoConn.Close()

	r := mux.NewRouter()

	r.HandleFunc("/login", login).Methods("POST")
	r.HandleFunc("/logout", logout).Methods("GET")
	r.HandleFunc("/healthcheck", healthCheck).Methods("GET")

	s := &http.Server{
		Addr:           ":8080",
		Handler:        r,
		ReadTimeout:    15 * time.Second,
		WriteTimeout:   15 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	log.Fatal(s.ListenAndServe())
}
