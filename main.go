package main

import (
	"bufio"
	"context"
	"fmt"
	"github.com/julienschmidt/httprouter"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"syscall"
	"time"
)

var (
	moscow *time.Location // время
)

const (
	port = "3000"
)

type Page struct {
	File string
	Body []byte
}

func (p *Page) save() error {
	path := strings.Join([]string{"read", p.File}, "/")
	return ioutil.WriteFile(path, p.Body, 0600)
}

func inlineLog(hiddenString, stringToLog string) string {
	log.Println(hiddenString, stringToLog)
	return stringToLog
}

func faviconHandler(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	http.ServeFile(w, r, inlineLog("└ Файл будет передан по запросу:", "favicon.ico"))
}

func create(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	http.ServeFile(w, r, "editor.html")
}

func save(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

		var pageData Page
		pageData.File = r.FormValue("filename")
		pageData.Body = []byte(r.FormValue("content"))
		fmt.Println("Новый файл будет создан", pageData.File)

		err := pageData.save()
		if err != nil {
			log.Println(err)
			return
		}
		fmt.Println("Файл был записан")

		_, err = fmt.Fprintf(w, "<a href=/read/%s>Открыть новый файл (%s)</a>", pageData.File, pageData.File)
		if err != nil {
		    log.Println(err)
		}
}

func main() {

	// объявляем роутер
	var m *Middleware
	m = newMiddleware(
		httprouter.New(),
	)

	webServer := http.Server{
		Addr:    net.JoinHostPort("", port),
		Handler: m,
		//TLSConfig:         nil,
		ReadTimeout:       1 * time.Minute,
		ReadHeaderTimeout: 15 * time.Second,
		WriteTimeout:      1 * time.Minute,
		//IdleTimeout:       0,
		//MaxHeaderBytes:    0,
		//TLSNextProto:      nil,
		//ConnState:         nil,
		//ErrorLog:          nil,
		//BaseContext:       nil,
		//ConnContext:       nil,
	}

	m.router.ServeFiles("/read/*filepath", http.Dir("read"))
	m.router.ServeFiles("/images/*filepath", http.Dir("images"))
	m.router.ServeFiles("/editor/*filepath", http.Dir("editor"))

	m.router.GET("/favicon.ico", faviconHandler)

	m.router.GET("/create/", create)
	m.router.POST("/save/", save)
	m.router.POST("/upload/", uploadFile)
	m.router.GET("/", index)

	var err error

	fmt.Println("Launching the service on the port:", port, "...")
	go func() {
		err = webServer.ListenAndServe()
		if err != nil {
			log.Println(err)
		}
	}()

	fmt.Println("The server was launched!")

	interrupt := make(chan os.Signal, 1)
	signal.Notify(interrupt, os.Interrupt, syscall.SIGTERM)

	<-interrupt

	timeout, cancelFunc := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancelFunc()

	err = webServer.Shutdown(timeout)
	if err != nil {
		log.Println(err)
	}
}

func index(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	http.ServeFile(w, r, "index.html")
}

func uploadFile(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {

	// Максимальный размер файлов 10 Мб
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20+512)
	reader, err := r.MultipartReader() // Разбираем multipartform
	if err != nil {
		log.Println("└", err)
		_, err = fmt.Fprintln(w, err)
		if err != nil {
			log.Println(err)
		}
		return
	}

	for { // цикл работы с файлами, один за одним

		// file — это FileHeader из которого мы
		// получаем имя файла, заголовок и его размер
		file, err := reader.NextPart()

		// Сначала проверяем на конец файла в текущей итерации
		if err == io.EOF {
			log.Println("└ The data array is depleted. Finished processing data")
			break // выходим из цикла
		}

		// А потом на другие ошибки проверяем
		if err != nil {
			log.Println(err)
			return // и завершаем работу хандлера
		}

		// Проверяем что это файл
		// log.Println(file)
		if file.FileName() == "" {
			log.Println("└ The data is not a file. Passed")
			continue // и пропускаем данные если названия файла нет
		}

		fmt.Println("└", file.FileName(), "has been successfully retrieved from the request header")

		fmt.Println("  └ Announced by browser Content-Type:", file.Header["Content-Type"][0])

		// Вынимаем реальный MIME Content-type
		buf := bufio.NewReader(file)
		sniff, _ := buf.Peek(512)
		contentType := http.DetectContentType(sniff)
		fmt.Println("  └ Real Content-Type:", contentType)

		fileNameParts := make([]string, 2) // переменная для формирования нового названия файла
		switch {                           // устанавливаем новое расширение для png-, gif- и jpg-файлов
		case contentType == "image/png":
			fileNameParts[1] = "png"
		case contentType == "image/gif":
			fileNameParts[1] = "gif"
		case contentType == "image/jpeg":
			fileNameParts[1] = "jpg"
		// проверяем SVG-файлы
		case strings.Contains(contentType, "text/xml") && file.Header["Content-Type"][0] == "image/svg+xml":
			fileNameParts[1] = "svg"
			contentType = file.Header["Content-Type"][0]
		case strings.Contains(contentType, "text/plain") && file.Header["Content-Type"][0] == "image/svg+xml" && strings.Contains(string(sniff), "<svg"):
			fileNameParts[1] = "svg"
			contentType = file.Header["Content-Type"][0]
		default: // завершаем MIME Content-type неверный
			log.Println("  └ Restricted Content-Type. The file will not be saved")
			continue
		}

		fmt.Println("  └", file.FileName(), "has been successfully passed the MIME-type check")

		// вынимаем расширение из файла чтобы его отрезать
		re := regexp.MustCompile(`\.(?i)(png|jpg|gif|jpeg|svg)$`) // переменная re содержит регулярку
		extension := string(re.Find([]byte(file.FileName())))     // формируем расширение по шаблону выше
		if extension == "" {
			log.Println("  └ The file's extension is wrong. The file will not be saved")
			continue
		}
		fmt.Println("  └", file.FileName(), "has a correct extension")

		// Проверяем что MIME-type из файла из из браузера соответствуют друг другу, исключение - SVG
		if file.Header["Content-Type"][0] != contentType {
			log.Println("  └ The MIME-types of the file do not match. The file will not be saved")
			continue
		}

		// Формируем строку для ioutil.TempFile
		fileNameParts[0] = strings.TrimSuffix(file.FileName(), extension) // filename.
		templateFilename := strings.Join(fileNameParts, "*.")             // filename*.extension
		log.Println("  └", fileNameParts[1], "is the file extention.", fileNameParts[0], "is the filename")

		// Временный файл с добавленными случайными цифрами перед расширением создаётся в папке "temp-images"
		tempFile, err := ioutil.TempFile("images", templateFilename)
		defer tempFile.Close() // Отложенная операция закрытия файла
		log.Println("  └", templateFilename, "is the template to name the uploaded file")
		if err != nil {
			log.Println(err)
		}

		// Вычитываем файл в виде массива битов
		fileBytes, err := ioutil.ReadAll(buf)
		if err != nil {
			log.Println(err)
		}

		// log.Println("  └ Размер файла: ? Kбайт") // Размер пока не знаю как показать

		// Записываем массив битов в tempFile
		_, err = tempFile.Write(fileBytes)
		if err != nil {
			log.Println(err)
		} else {
			// Сообщаем пользователю об успехе!
			fmt.Println("  └", tempFile.Name(), "is the file path")
			newFileName := tempFile.Name()[7:]
			fmt.Println("  └", file.FileName(), "has been successfully saved. The new name is", newFileName)
			_, err = fmt.Fprintf(w, `{
	"uploaded": true,
	"url": "/images/%s"
}
			`, newFileName)
			if err != nil {
				log.Println(err)
			}
		}
	}
}
