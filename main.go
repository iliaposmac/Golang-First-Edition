package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"html/template"

	"github.com/go-redis/redis"
	"github.com/gorilla/mux"
	"github.com/gorilla/sessions"
)

var PORT int = 3000
var client *redis.Client

var store = sessions.NewCookieStore([]byte("SESSION_SECRET_KEY"))

func main() {

	client = redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	router := mux.NewRouter()

	router.HandleFunc("/", homePage).Methods("GET")
	router.HandleFunc("/comments", comments).Methods("GET")
	router.HandleFunc("/comments", createNewComment).Methods("POST")
	router.HandleFunc("/login", getLoginPage).Methods("GET")
	router.HandleFunc("/login", loginPostPage).Methods("POST")

	fileServer := http.FileServer(neuteredFilsSystem{http.Dir("./static/")})

	router.Handle("/static/**", http.NotFoundHandler())

	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fileServer))

	http.Handle("/", router)

	infoLogs("Запуск веб-сервера на http://localhost:" + fmt.Sprint(PORT))

	serverStructure := &http.Server{
		Addr:    ":3000",
		Handler: router,
	}

	err := serverStructure.ListenAndServe()
	if err != nil {
		fatalLogs(err, "Can not start server")
	}
}

type neuteredFilsSystem struct {
	fs http.FileSystem
}

func (nfs neuteredFilsSystem) Open(path string) (http.File, error) {
	f, err := nfs.fs.Open(path)
	if err != nil {
		return nil, err
	}

	s, err := f.Stat()
	if s.IsDir() {
		index := filepath.Join(path, "index.html")
		if _, err := nfs.fs.Open(index); err != nil {
			closeErr := f.Close()
			if closeErr != nil {
				return nil, closeErr
			}

			return nil, err
		}
	}

	return f, nil
}

func homePage(w http.ResponseWriter, req *http.Request) {

	if req.URL.Path != "/" {
		http.NotFound(w, req)
		return
	}
	showUrlLogs(req)

	files := []string{
		"./pages/home.page.html",
		"./pages/layouts/base.layout.html",
		"./pages/html/header.partial.html",
		"./pages/html/nav.partial.html",
		"./pages/html/footer.partial.html",
	}

	ts, err := template.ParseFiles(files...)

	if err != nil {
		warnings(string(err.Error()))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	err = ts.Execute(w, nil)

	if err != nil {
		log.Println(err.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func comments(w http.ResponseWriter, req *http.Request) {

	showUrlLogs(req)

	files := []string{
		"./pages/comments.page.html",
		"./pages/layouts/base.layout.html",
		"./pages/html/header.partial.html",
		"./pages/html/nav.partial.html",
		"./pages/html/footer.partial.html",
	}

	ts, tempalteError := template.ParseFiles(files...)

	if tempalteError != nil {
		warnings(string(tempalteError.Error()))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	pageComments, redisClientError := client.LRange("comments", 0, 10).Result()

	if redisClientError != nil {
		return
	}

	tsExecError := ts.Execute(w, pageComments)

	if tsExecError != nil {
		log.Println(tsExecError.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func createNewComment(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	comment := req.PostForm.Get("comment")
	showUrlLogs(req)
	client.LPush("comments", comment)
	http.Redirect(w, req, "/comments", http.StatusFound)
}

func getLoginPage(w http.ResponseWriter, req *http.Request) {
	showUrlLogs(req)

	files := []string{
		"./pages/login.page.html",
		"./pages/layouts/base.layout.html",
		"./pages/html/header.partial.html",
		"./pages/html/nav.partial.html",
		"./pages/html/footer.partial.html",
	}

	ts, tempalteError := template.ParseFiles(files...)

	if tempalteError != nil {
		warnings(string(tempalteError.Error()))
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	value, status := getFromSession(req, "username")

	if status == 0 {
		//
	}
	infoLogs(value)
	tsExecError := ts.Execute(w, nil)

	if tsExecError != nil {
		log.Println(tsExecError.Error())
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

func loginPostPage(w http.ResponseWriter, req *http.Request) {
	req.ParseForm()
	username := req.PostForm.Get("username")
	showUrlLogs(req)
	session, _ := store.Get(req, "session")
	session.Values["username"] = username
	session.Save(req, w)
	http.Redirect(w, req, "/comments", http.StatusFound)
}

func getFromSession(req *http.Request, key string) (value string, status int) {
	session, _ := store.Get(req, "session")
	untyped, ok := session.Values[key]

	if !ok {
		return "", 0
	}

	getValue, ok := untyped.(string)

	if !ok {
		return "", 0
	}
	return getValue, 1
}

func showUrlLogs(req *http.Request) {
	infoLogs("User accessed " + req.Method + " " + req.URL.Path)
	infoLogs("Query Search Params: " + string(req.URL.Query().Get("search")))
	infoLogs("Query Filter Params: " + string(req.URL.Query().Get("filter")))
}

func infoLogs(text string) {
	infoLog := log.New(writeLogs(), "[ INFO ]\t", log.Ldate|log.Ltime)
	infoLog.SetOutput(writeLogs())
	infoLog.Println(text)
	fmt.Println(createLogsWithTime("[ INFO ]", text))
}

func fatalLogs(err error, text string) {
	errorLog := log.New(writeLogs(), "[ ERROR ]\t", log.Ldate|log.Ltime|log.Llongfile|log.Lmicroseconds)
	errorLog.SetOutput(writeLogs())
	errorLog.Fatalln("Fatal error: ", text, err.Error())
	fmt.Println(createLogsWithTime("[ ERROR ]", text))
}

func warnings(text string) {
	warnLog := log.New(writeLogs(), "[ WANR ]\t", log.Ldate|log.Ltime)
	warnLog.SetOutput(writeLogs())
	warnLog.Println(text)
	fmt.Println(createLogsWithTime("[ WANR ]", text))
}

func writeLogs() *os.File {
	f, err := os.OpenFile("logs.log", os.O_RDWR|os.O_CREATE, 0666)
	if err != nil {
		fatalLogs(err, "Can not write logs to info.log")
	}
	defer f.Close()
	return f
}

func createLogsWithTime(level string, text string) string {
	timeStamp := fmt.Sprint(log.Ldate) + "  " + fmt.Sprint(log.Ltime)
	return timeStamp + " " + level + " " + text
}
