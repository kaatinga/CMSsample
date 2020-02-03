package main

import (
	"bufio"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"
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

func faviconHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("-------------------", time.Now().Format(http.TimeFormat), "A favicon.ico request is received -------------------")
	log.Println("На сервер обратился клиент с адреса", r.RemoteAddr)
	http.ServeFile(w, r, inlineLog("└ Файл будет передан по запросу:", "favicon.ico"))
}

func create(w http.ResponseWriter, r *http.Request) {
	log.Println("-------------------", time.Now().Format(http.TimeFormat), "A common request is received -------------------")
	log.Println("На сервер обратился клиент с адреса", r.RemoteAddr)
	log.Println("Открываем редактор...")
	http.ServeFile(w, r, "editor.html")
}

func save(w http.ResponseWriter, r *http.Request) {
	log.Println("-------------------", time.Now().Format(http.TimeFormat), "A common request is received -------------------")
	log.Println("На сервер обратился клиент с адреса", r.RemoteAddr)
	log.Println("Проверяем метод...")
	if r.Method == "POST" {
		var pageData Page
		pageData.File = r.FormValue("filename")
		pageData.Body = []byte(r.FormValue("content"))
		log.Println("Новый файл будет создан", pageData.File)
		err := pageData.save()
		if err != nil {
		    log.Println(err)
		} else {
			log.Println("Файл был записан")
		}
		fmt.Fprintf(w, "<a href=/read/%s>Открыть новый файл (%s)</a>", pageData.File, pageData.File)
	} else {
		log.Println("Неверный метод...")
		http.Error(w, http.StatusText(405), 405)
	}

}

func main() {

	http.Handle("/read/", http.StripPrefix("/read/", http.FileServer(http.Dir("read"))))
	http.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir("images"))))
	http.Handle("/editor/", http.StripPrefix("/editor/", http.FileServer(http.Dir("editor"))))

	http.HandleFunc("/create/", create)
	http.HandleFunc("/save/", save)
	http.HandleFunc("/upload/", uploadFile)
	http.HandleFunc("/", index)
	http.HandleFunc("/favicon.ico", faviconHandler) // Обработчик favicon.ico


	log.Println("Server starting...")
	http.ListenAndServe(":3000", nil)
}

func index(w http.ResponseWriter, r *http.Request) {
	log.Println("-------------------", time.Now().Format(http.TimeFormat), "A common request is received -------------------")
	log.Println("На сервер обратился клиент с адреса", r.RemoteAddr)
	log.Println("Открываем главную страницу...")
	http.ServeFile(w, r, "index.html")
}

func uploadFile(w http.ResponseWriter, r *http.Request) {
	log.Println("-------------------", time.Now().Format(http.TimeFormat), "File upload request is received -------------------")
	log.Println("На сервер обратился клиент с адреса", r.RemoteAddr)

	if r.Method != "POST" {
		log.Println("└ Метод обращения не верный...")
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	// Максимальный размер файлов 10 Мб
	r.Body = http.MaxBytesReader(w, r.Body, 10<<20+512)
	reader, err := r.MultipartReader() // Разбираем multipartform
	if err != nil {
		fmt.Fprintln(w, err)
		log.Println("└ Нет файлов...")
		return
	}

	for { // цикл работы с файлами, один за одним

		// file — это FileHeader из которого мы
		// получаем имя файла, заголовок и его размер
		file, err := reader.NextPart()

		// Сначала проверяем на конец файла в текущей итерации
		if err == io.EOF {
			break // выходим из цикла
		}

		// А потом на другие ошибки проверяем
		if err != nil {
			log.Println(err)
			return // и завершаем работу хандлера
		}

		// Проверяем название файла
		if file.FileName() == "" {
			log.Println("Zero file detected. Finished processing files")
			return // и завершаем работу хандлера
		}

		log.Println("└", file.FileName(), "has been successfully retrieved form the request header")

		log.Println("  └ Заявленный Content-Type:", file.Header["Content-Type"][0])

		// Проверяем MIME Content-type
		buf := bufio.NewReader(file)
		sniff, _ := buf.Peek(512)
		contentType := http.DetectContentType(sniff)
		log.Println("  └ Реальный Content-Type:", contentType)
		re := regexp.MustCompile(`image/(png|gif|jpeg)`) // переменная re содержит регулярку
		MIMEtype := re.Find([]byte(contentType))
		if MIMEtype == nil {
			log.Println("  └ Restricted Content-Type. The file will not be saved")
			continue
		}
		log.Println("  └", file.FileName(), "has been successfully passed the MIME-type check")

		re = regexp.MustCompile(`\.(?i)(png|jpg|gif|jpeg)$`)    // меняем шаблон регулярки !!! ТОЧКА НЕ ПРОВЕРЯЕТСЯ ПОКА !!!
		extension := string(re.Find([]byte(file.FileName()))) // формируем расширение по шаблону выше
		if extension == "" {
			log.Println("  └ The file's extension is wrong. The file will not be saved")
			continue
		}
		log.Println("  └", file.FileName(), "has an allowed extension")

		// Проверяем расширение файла и создаём новую переменную extensionToCheck для проверки MIME-type
		var extensionToCheck string
		switch {
		case strings.ToLower(extension) == ".jpg":
			extensionToCheck = "jpeg"
		default:
			extensionToCheck = strings.ToLower(extension[1:])
		}

		log.Println("  └", extensionToCheck, "is extensionToCheck")

		// Проверяем что MIME-type соответствует расширению
		re = regexp.MustCompile(`(?i)(png|gif|jpeg)`)
		if string(re.Find([]byte(contentType))) != extensionToCheck {
			log.Println("  └ The MIME-type and extension of the file do not match. The file will not be saved")
			continue
		}

		// Формируем строку для ioutil.TempFile
		fileNameParts := make([]string, 2)
		fileNameParts[1] = extension[1:]                                  // .extension, убираем точку заодно
		fileNameParts[0] = strings.TrimSuffix(file.FileName(), extension) // filename.
		templateFilename := strings.Join(fileNameParts, "*.")             // filename*.extension
		log.Println("  └", fileNameParts[1], "is extention.",fileNameParts[0],"is name")

		// Временный файл с добавленными случайными цифрами перед расширением создаётся в папке "temp-images"
		tempFile, err := ioutil.TempFile("images", templateFilename)
		defer tempFile.Close() // Отложенная операция закрытия файла
		log.Println("  └", templateFilename,"is template for the file name")
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
			log.Println("  └", tempFile.Name(),"is path to file")
			newFileName := tempFile.Name()[7:]
			log.Println("  └", file.FileName(), "has been successfully saved. The new name is", newFileName)
			fmt.Fprintf(w, `{
	"uploaded": true,
	"url": "/images/%s"
}
			`, newFileName)
		}
	}
}


