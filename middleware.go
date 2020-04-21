package main

import (
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
	"time"
)

// Middleware wraps julien's router http methods
type Middleware struct {
	router *httprouter.Router
}

// newMiddleware returns pointer of Middleware
func newMiddleware(r *httprouter.Router) *Middleware {
	return &Middleware{r}
}

// мидлвейр для всех хэндлеров
func (rw *Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	log.Println("-------------------", time.Now().In(moscow).Format(http.TimeFormat), "A request is received -------------------")
	log.Println("The request is from", r.RemoteAddr, "| Method:", r.Method, "| URI:", r.URL.String())

	if r.Method == "POST" {
		// проверяем размер POST данных
		r.Body = http.MaxBytesReader(w, r.Body, 10000)
		err := r.ParseForm()
		if err != nil {
			log.Println("POST data is exceeded the limit")
			http.Error(w, http.StatusText(400), 400)
			return
		}
	}

	rw.router.ServeHTTP(w, r)
}
