package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

func HttpCookies(w http.ResponseWriter, r *http.Request) {
	/**
		cName := http.Cookie{
			Name:       "name",      // имя куки
			Value:      "golang",    // Значение, соответствующее имени файла cookie
			Path:       "/",         //
			Domain:     "",          // Объем cookie
			Expires:    time.Time{}, // срок действия cookie
			RawExpires: "",
			MaxAge:     0,     // Устанавливаем время истечения, соответствующее атрибуту MaxAge файла cookie браузера
			Secure:     false, // Установите атрибут Secure (Примечание: атрибут Secure Cookie означает, что обмен файлами cookie ограничен зашифрованной передачей, что указывает на то, что браузер может использовать cookie только через безопасное / зашифрованное соединение. Если веб-сервер настроен из небезопасного соединение Файл cookie с безопасным атрибутом, когда файл cookie отправляется клиенту, он все еще может быть перехвачен атакой типа `` человек посередине '')
			HttpOnly:   true,  // Устанавливаем атрибут httpOnly (Примечание: атрибут HttpOnly файла cookie указывает браузеру не предоставлять файл cookie, за исключением запросов HTTP (и HTTPS). К файлу cookie с атрибутом HttpOnly нельзя получить доступ с помощью методов, отличных от HTTP, например с помощью вызов JavaScript (например, ссылка document.cookie), поэтому невозможно украсть этот файл cookie с помощью междоменного сценария (очень распространенный метод атаки). В частности, Facebook и Google широко используют атрибут HttpOnly.)
			SameSite:   0,
			Raw:        "",
			Unparsed:   nil,
		}
		cId := http.Cookie{
			Name:       "id",
			Value:      "21",
			Path:       "/",
			Domain:     "",
			Expires:    time.Time{},
			RawExpires: "",
			MaxAge:     0,
			Secure:     false,
			HttpOnly:   true,
			SameSite:   0,
			Raw:        "",
			Unparsed:   nil,
		}
		// установить cookie
		w.Header().Set("Set-Cookie", cId.String())
		w.Header().Add("Set-Cookie", cName.String())
		// установить cookie
		http.SetCookie(w, &http.Cookie{
			Name:       "mobile",
			Value:      "13388888888",
			Path:       "/",
			Domain:     "",
			Expires:    time.Time{},
			RawExpires: "",
			MaxAge:     0,
			Secure:     false,
			HttpOnly:   true,
			SameSite:   0,
			Raw:        "",
			Unparsed:   nil,
		})
		http.SetCookie(w, &http.Cookie{
			Name:       "email",
			Value:      "golang@126.cn",
			Path:       "/",
			Domain:     "",
			Expires:    time.Time{},
			RawExpires: "",
			MaxAge:     0,
			Secure:     false,
			HttpOnly:   true,
			SameSite:   0,
			Raw:        "",
			Unparsed:   nil,
		})
		// читаем cookie
		name := r.Header["Cookie"]
		fmt.Println(name)
		// Получаем куки по ключу
		id, err := r.Cookie("id")
		if err != nil {
			fmt.Println(err.Error())
		} else {
			fmt.Println(id)
		}
		// Получить все куки
		cookies := r.Cookies()
		for i, c := range cookies {
			fmt.Println("кука ", i, " : ", c)
		}
	**/
}

func readCook(w http.ResponseWriter, r *http.Request) {
	// Получить все куки
	cookies := r.Cookies()
	for i, c := range cookies {
		fmt.Println("кука ", i, " : ", c)
	}
}

func keyCook(w http.ResponseWriter, r *http.Request) {
	// Получаем куки по ключу
	name, err := r.Cookie("name")
	if err != nil {
		fmt.Println("такой куки нет")
		fmt.Println(err.Error())
	} else {
		fmt.Println("name = ", name.Value)
	}
	name, err = r.Cookie("name2")
	if err != nil {
		fmt.Println(err.Error())
	} else {
		fmt.Println("name2 = ", name.Value)
	}
}

func setCook(w http.ResponseWriter, r *http.Request) {
	name := "Rick"
	cName := http.Cookie{
		Name:  "name", // имя куки
		Value: name,   // значение
	}
	w.Header().Set("Set-Cookie", cName.String())

	name2 := "Nick"
	cName2 := &http.Cookie{
		Name:  "name2",
		Value: name2,
	}
	http.SetCookie(w, cName2)
}

func delCook(w http.ResponseWriter, r *http.Request) {
	c := &http.Cookie{
		Name:    "name",
		Value:   "",
		Path:    "/",
		Expires: time.Unix(0, 0),

		HttpOnly: true,
	}
	http.SetCookie(w, c)
}

func main() {
	http.HandleFunc("/", HttpCookies)
	http.HandleFunc("/read", readCook)
	http.HandleFunc("/key", keyCook)
	http.HandleFunc("/set", setCook)
	http.HandleFunc("/del", delCook)
	log.Println(http.ListenAndServe(":5006", nil))
}
