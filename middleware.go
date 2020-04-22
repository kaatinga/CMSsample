package main

import (
	"context"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"log"
	"net/http"
	"time"
)

type Adapter func(httprouter.Handle) httprouter.Handle

func Adapt(next httprouter.Handle, adapters ...Adapter) httprouter.Handle {
	for _, adapter := range adapters {
		next = adapter(next)
	}
	return next
}

func Wrapper() Adapter {
	return func(next httprouter.Handle) httprouter.Handle {
		return func(w http.ResponseWriter, r *http.Request, actions httprouter.Params) {
			var rd ViewData

			// рендер работает отложенно с проверкой условия
			defer func() {
				if rd.render {
					Render(w, &rd)
				}
			}()

			ctx := context.WithValue(r.Context(), "rd", &rd)
			next(w, r.WithContext(ctx), actions)
		}
	}
}

// Middleware wraps julien's router http methods
type Middleware struct {
	router *httprouter.Router
}

// newMiddleware returns pointer of Middleware
func newMiddleware(r *httprouter.Router) *Middleware {
	return &Middleware{r}
}

// мидлвейр для всех хэндлеров
func (m *Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Println("-------------------", time.Now().In(moscow).Format(http.TimeFormat), "A request is received -------------------")
	fmt.Println("The request is from", r.RemoteAddr, "| Method:", r.Method, "| URI:", r.URL.String())

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

	m.router.ServeHTTP(w, r)
}
