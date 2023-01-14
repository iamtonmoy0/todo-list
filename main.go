package main

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"time"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/thedevsaddam/renderer"
	"gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

var rnd *renderer.Render

var db *mgo.Database

const (
	hostName       string = "localhost:27017"
	dbName         string = "demo_todo"
	collectionName string = "todo"
	port           string = ":9000"
)

type (
	todoModel struct {
		ID        bson.ObjectId `bson:"_id,omitempty"`
		Title     string        `bson:"title"`
		Completed bool          `bson:"completed"`
		CreateAt  time.Time     `bson:"createAt"`
	}

	todo struct {
		ID        string    `json:"id"`
		Title     string    `json:"title"`
		Completed string    `json:"complete"`
		CreateAt  time.Time `json:"create_at"`
	}
)

func init() {
	rnd = renderer.New()
	sess, err := mgo.Dial(hostName)
	checkErr(err)
	sess.SetMode(mgo.Monotonic, true)
	db = sess.DB(dbName)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {

	err := rnd.Template(w, http.StatusOK, []string{"/static/home.tpl"}, nil)
	checkErr(err)
}
func fetchTodos(w http.ResponseWriter, r *http.Request) {
	todo := []todoModel{}

	if err := db.C(collectionName).Find(bson.M{}).All(&todo); err != nil {
		rnd.JSON(w, http.StatusProcessing, renderer.M{
			"message": "failed to fetch todo",
			"error":   err,
		})
		return
	}
	todoList := []todo{}

	for _, t := range todos {
		todoList = append(todoList, todo{
			ID:        t.ID.Hex(),
			Title:     t.Title,
			Completed: t.completed,
			CreatedAt: t.createdAt,
		})
	} //bson data from db and send to frontend ,
	rnd.JSON(w, http.StatusOK, renderer.M{
		"data": todoList,
	})
}

func createTodo(w http.ResponseWriter, r *http.Request) {
	var t todo
	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		rnd.JSON(w, http.StatusProcessing, err)
		return
	}
	// simple validation
	if t.Title == "" {
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message": "title field is required",
		})
		return
	}

	// if input is valid ,then create todo
	tm := todoModel{
		ID:        bson.NewObjectId(),
		Title:     t.Title,
		Completed: false,
		CreateAt:  time.Now(),
	}
	if err := db.C(collectionName).Insert(&tm); err != nil {
		rnd.JSON(w, http.StatusProcessing, renderer.M{
			"message": "failed to same todo",
			"error":   err,
		})
		return
	}
	rnd.JSON(w, http.StatusCreated, renderer.M{
		"message": "todo created success",
		"todo_id": tm.ID.Hex(),
	})
}
func deleteTodo(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))

	if !bson.IsObjectIdHex(id) {
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message": "The id is invalid",
		})
		return
	}

	if err := db.C(collectionName).RemoveId(bson.ObjectIdHex(id)); err != nil {
		rnd.JSON(w, http.StatusProcessing, renderer.M{
			"message": "Failed to delete todo",
			"error":   err,
		})
		return
	}

	rnd.JSON(w, http.StatusOK, renderer.M{
		"message": "Todo deleted successfully",
	})
}
func updateTodo(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(chi.URLParam(r, "id"))
	if !bson.IsObjectIdHex(id) {
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message": "the id is invalid",
		})
		return
	}

	var t todo

	if err := json.NewDecoder(r.Body).Decode(&t); err != nil {
		rnd.JSON(w, http.StatusProcessing, err)
		return
	}
	if t.Title == "" {
		rnd.JSON(w, http.StatusBadRequest, renderer.M{
			"message": "the title field is required",
			//just for validation
		})
		return
	}

	if err := db.C(collectionName).Update(
		bson.M{"_id": bson.ObjectIdHex(id)},
		bson.M{"title": t.Title, "completed": t.Completed},
	); err != nil {
		rnd.JSON(w, http.StatusProcessing, renderer.M{
			"message": "failed to update todo",
			"error":   err,
		})
		return
	}
}

func main() {
	stopChan := make(chan os.Signal) //for stopping server
	signal.Notify(stopChan, os.Interrupt)
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/", homeHandler)
	r.Mount("/", todoHandlers())
	srv := &http.Server{
		Addr:         port,
		Handler:      r,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  60 * time.Second,
	}
	go func() {
		log.Println("listening on port ", port)
		if err := srv.ListenAndServe(); err != nil {
			log.Printf("listen:%s\n", err)
		}
	}()

	<-stopChan
	log.Println("shutting down server ........")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	srv.Shutdown(ctx)
	defer cancel()
	log.Println("server gracefully stopped....!")

}

func todoHandlers() http.Handler {
	rg := chi.NewRouter()
	rg.Group(func(r chi.Router) {
		r.Get("/", fetchTodos)
		r.Post("/", createTodo)
		r.Put("/", updateTodo)
		r.Delete("/", deleteTodo)
	})
}
func checkErr(err error) {
	if err != nil {
		log.Fatal(err)
	}

}
