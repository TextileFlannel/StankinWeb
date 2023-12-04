package main

import (
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"
	"strings"

	"github.com/gorilla/sessions"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

var store = sessions.NewCookieStore([]byte("secret-key"))

type NewsItem struct {
	Id int
	Title string
	Text string
	Date string
}

type FeedbackItem struct{
	Id int
	Name string
	Email string
	Text string
}

type SupItem struct {
	Id int
	Image string
	Name string
	Short string
	Long string
}

type RentItem struct {
	Id int
	Name string
	Square int
	Price int
	Image string
	Phone string
	Email string
	About string
	Ind int
	InBasket string
}

type SaleItem struct {
	Id int
	Title string
	Text string
}

func registr(w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open("sqlite3", "my_db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
	session, err := store.Get(r, "user-session")
	_, ok := session.Values["user"]
	if ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}else if r.Method == http.MethodPost {
		tmpl, err := template.ParseFiles("templates/registration.html")
		if err != nil{
			fmt.Fprintf(w, err.Error())
		}
		name := r.FormValue("username")
		email := r.FormValue("email")
		pass := r.FormValue("password")
		if name == "admin"{
			data := struct{ Error string}{"Запрещенное имя пользователя"}
			tmpl.Execute(w, data)
			return
		}
		var check_email, check_name string
		er1 := db.QueryRow("SELECT email FROM users WHERE email = ?", email).Scan(&check_email)
		er2 := db.QueryRow("SELECT name FROM users WHERE name = ?", name).Scan(&check_name)

		if er1 == nil{
			data := struct{ Error string}{"Этот Email уже занят"}
			tmpl.Execute(w, data)
			return
		}
		if er2 == nil{
			data := struct{ Error string}{"Это имя пользователя уже занято"}
			tmpl.Execute(w, data)
			return
		}

		if r.FormValue("password") != r.FormValue("confirm_password"){
			data := struct{ Error string}{"Пароли не совпадают"}
			tmpl.Execute(w, data)
			return
		}
		passwordHash, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
		if err != nil {
			log.Fatal(err)
			http.Error(w, "Ошибка при хешировании пароля", http.StatusInternalServerError)
			return
		}
		_, err = db.Exec("INSERT INTO users (name, email, password) VALUES (?, ?, ?)", name, email, passwordHash)
		if err != nil {
			log.Fatal(err)
			http.Error(w, "Ошибка при сохранении данных в базу данных", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}else{
		tmpl, err := template.ParseFiles("templates/registration.html")
		if err != nil{
			fmt.Fprintf(w, err.Error())
		}
		tmpl.Execute(w, nil)
	}
}

func index(w http.ResponseWriter, r *http.Request) {
    tmpl, _ := template.ParseFiles("templates/index.html")
	session, err := store.Get(r, "user-session")
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
	user, ok := session.Values["user"]
    if ok {
        tmpl, _ := template.ParseFiles("templates/index.html")
        data := struct{ 
            Error string 
            User string
        }{User: user.(string), Error: ""}
        tmpl.Execute(w, data)
	}else{
		tmpl.Execute(w, nil)
	}
}

func login(w http.ResponseWriter, r *http.Request) {
    db, err := sql.Open("sqlite3", "my_db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    session, err := store.Get(r, "user-session")
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }

    _, ok := session.Values["user"]
    if ok {
		http.Redirect(w, r, "/", http.StatusSeeOther)
    }else if r.Method == http.MethodPost {
        tmpl, _ := template.ParseFiles("templates/login.html")
        name := r.FormValue("username")
        password := r.FormValue("password")
        
        var id, email, storedName, storedPassword string
        er := db.QueryRow("SELECT id, email, name, password FROM users WHERE name = ?", name).Scan(&id, &email, &storedName, &storedPassword)
        if er != nil{
            log.Println("Incorrect password for user: ", name)
            data := struct{ 
                Error string 
                User string
            }{User: "", Error: "Неверное имя пользователя или пароль"}
            tmpl.Execute(w, data)
			return
        }
        
        er = bcrypt.CompareHashAndPassword([]byte(storedPassword), []byte(password))
        if er != nil {
            log.Println("Incorrect password for user:", name)
            data := struct{ 
                Error string 
                User string
            }{User: "", Error: "Неверное имя пользователя или пароль"}
            tmpl.Execute(w, data)
            return
        }
        session.Values["user"] = name
		session.Values["id"] = id
		session.Values["email"] = email
        session.Save(r, w)
        http.Redirect(w, r, "/", http.StatusSeeOther)
    }else{
		tmpl, _ := template.ParseFiles("templates/login.html")
		tmpl.Execute(w, nil)
	}
}

func logout(w http.ResponseWriter, r *http.Request) {
    session, err := store.Get(r, "user-session")
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    session.Options.MaxAge = -1
    err = session.Save(r, w)
    if err != nil {
        http.Error(w, err.Error(), http.StatusInternalServerError)
        return
    }
    http.Redirect(w, r, "/", http.StatusSeeOther)
}

func card(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/card.html")
	if err != nil{
		fmt.Fprintf(w, err.Error())
	}
	db, err := sql.Open("sqlite3", "my_db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
	data := struct{ 
		Error string 
		User string
		Flag string
		News []NewsItem
	}{Error: "", Flag: ""}
	session, err := store.Get(r, "user-session")
	user, ok := session.Values["user"]
	if ok {
		db.QueryRow("SELECT id FROM cards WHERE user_id = ?", session.Values["id"]).Scan(&data.Flag)
		data.User = user.(string)
	} else{
		data.User = ""
	}
	tmpl.Execute(w, data)
}

func add_card(w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open("sqlite3", "my_db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

	session, _ := store.Get(r, "user-session")
	_, ok := session.Values["user"]
	if ok {
		num := r.FormValue("phone")
		db.Exec("INSERT INTO cards (num, user_id) VALUES (?, ?)", num, session.Values["id"])
		http.Redirect(w, r, "/", http.StatusSeeOther)
	} else{
		http.Redirect(w, r, "/login", http.StatusSeeOther)
	}
}

func news(w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open("sqlite3", "my_db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
	session, err := store.Get(r, "user-session")
	user, ok := session.Values["user"]
	rows, err := db.Query("SELECT id, title, text, date FROM news")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var news []NewsItem

	for rows.Next() {
		var n NewsItem
		if err := rows.Scan(&n.Id, &n.Title, &n.Text, &n.Date); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		news = append(news, n)
	}
	tmpl, _ := template.ParseFiles("templates/news.html")
	data := struct{
		Id int
		Error string 
		User string
		News []NewsItem
	}{Error: "", News: news}
	if ok {
		data.User = user.(string)
	} else{
		data.User = ""
	}
	tmpl.Execute(w, data)
}

func contacts(w http.ResponseWriter, r *http.Request) {
	tmpl, err := template.ParseFiles("templates/contacts.html")
	session, err := store.Get(r, "user-session")
	user, ok := session.Values["user"]
	if err != nil{
		fmt.Fprintf(w, err.Error())
	}

	db, err := sql.Open("sqlite3", "my_db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

	if r.Method == http.MethodPost{
		if ok{
			mes := r.FormValue("message")
			_, err = db.Exec("INSERT INTO feedback (name, email, mes) VALUES (?, ?, ?)", user, session.Values["email"], mes)
			if err != nil {
				log.Fatal(err)
				http.Error(w, "Ошибка при сохранении данных в базу данных", http.StatusInternalServerError)
				return
			}
			http.Redirect(w, r, "/", http.StatusSeeOther)
		}else{
			http.Redirect(w, r, "/login", http.StatusSeeOther)
		}
	}else{
		data := struct{ 
			Error string 
			User string
		}{Error: ""}
		
		if ok {
			data.User = user.(string)
		} else{
			data.User = ""
		}
		tmpl.Execute(w, data)
	}
}

func sale(w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open("sqlite3", "my_db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
	session, err := store.Get(r, "user-session")
	user, ok := session.Values["user"]
	rows, err := db.Query("SELECT id, title, text FROM sale")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var sale []SaleItem

	for rows.Next() {
		var n SaleItem
		if err := rows.Scan(&n.Id, &n.Title, &n.Text); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		sale = append(sale, n)
	}
	tmpl, _ := template.ParseFiles("templates/sale.html")
	data := struct{
		Id int
		Error string 
		User string
		Sale []SaleItem
	}{Error: "", Sale: sale}
	if ok {
		data.User = user.(string)
	} else{
		data.User = ""
	}
	tmpl.Execute(w, data)
}

func add_sale(w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open("sqlite3", "my_db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
	session, err := store.Get(r, "user-session")
	user, ok := session.Values["user"]
	if !ok || user != "admin"{
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}else if r.Method == http.MethodPost {
		title := r.FormValue("title")
		text := r.FormValue("text")
		_, err = db.Exec("INSERT INTO sale (title, text) VALUES (?, ?)", title, text)
		if err != nil {
			log.Fatal(err)
			http.Error(w, "Ошибка при сохранении данных в базу данных", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/account", http.StatusSeeOther)
	}else{
		tmpl, _ := template.ParseFiles("templates/add_sale.html")
		data := struct{ 
            Error string 
            User string
        }{User: user.(string), Error: ""}
		tmpl.Execute(w, data)
	}
}

func del_sale(w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open("sqlite3", "my_db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

	_, er := db.Exec("DELETE FROM sale WHERE id = ?", r.FormValue("sale_id"))
	if er != nil {
		http.Error(w, er.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/sale", http.StatusSeeOther)
}

func support(w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open("sqlite3", "my_db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
	session, err := store.Get(r, "user-session")
	user, ok := session.Values["user"]
	rows, err := db.Query("SELECT id, image, name, short, long FROM support")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var sup []SupItem

	for rows.Next() {
		var n SupItem
		if err := rows.Scan(&n.Id, &n.Image, &n.Name, &n.Short, &n.Long); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		sup = append(sup, n)
	}
	tmpl, _ := template.ParseFiles("templates/support.html")
	data := struct{
		Id int
		Error string 
		User string
		Sup []SupItem
	}{Error: "", Sup: sup}
	if ok {
		data.User = user.(string)
	} else{
		data.User = ""
	}
	tmpl.Execute(w, data)
}

func add_news(w http.ResponseWriter, r *http.Request){
	db, err := sql.Open("sqlite3", "my_db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
	session, err := store.Get(r, "user-session")
	user, ok := session.Values["user"]
	if !ok || user != "admin"{
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}else if r.Method == http.MethodPost {
		title := r.FormValue("title")
		text := r.FormValue("text")
		months := map[int]string{
			1:  "января",
			2:  "февраля",
			3:  "марта",
			4:  "апреля",
			5:  "мая",
			6:  "июня",
			7:  "июля",
			8:  "августа",
			9:  "сентября",
			10: "октября",
			11: "ноября",
			12: "декабря",
		}
		date_now := time.Now()
		date := fmt.Sprintf("%02d %s %d", date_now.Day(), months[int(date_now.Month())], date_now.Year())
		_, err = db.Exec("INSERT INTO news (title, text, date) VALUES (?, ?, ?)", title, text, date)
		if err != nil {
			log.Fatal(err)
			http.Error(w, "Ошибка при сохранении данных в базу данных", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/account", http.StatusSeeOther)
	}else{
		tmpl, _ := template.ParseFiles("templates/add_news.html")
		data := struct{ 
            Error string 
            User string
        }{User: user.(string), Error: ""}
		tmpl.Execute(w, data)
	}
}

func del_news(w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open("sqlite3", "my_db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

	_, er := db.Exec("DELETE FROM news WHERE id = ?", r.FormValue("news_id"))
	if er != nil {
		http.Error(w, er.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/news", http.StatusSeeOther)
}

func account(w http.ResponseWriter, r *http.Request) {
	session, _ := store.Get(r, "user-session")
	user, ok := session.Values["user"]
	if ok && user == "admin"{
		tmpl, err := template.ParseFiles("templates/account.html")
		if err != nil{
			fmt.Fprintf(w, err.Error())
		}
		data := struct{ 
            Error string 
            User string
        }{User: user.(string), Error: ""}
        tmpl.Execute(w, data)
	}else{
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
}

func add_support(w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open("sqlite3", "my_db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
	session, err := store.Get(r, "user-session")
	user, ok := session.Values["user"]
	if !ok || user != "admin"{
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}else if r.Method == http.MethodPost {
		image := "static/images/" + r.FormValue("image")
		name := r.FormValue("name")
		short := r.FormValue("short")
		long := r.FormValue("long")
		_, err = db.Exec("INSERT INTO support (image, name, short, long) VALUES (?, ?, ?, ?)", image, name, short, long)
		if err != nil {
			log.Fatal(err)
			http.Error(w, "Ошибка при сохранении данных в базу данных", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/account", http.StatusSeeOther)
	}else{
		tmpl, _ := template.ParseFiles("templates/add_sup.html")
		data := struct{ 
            Error string 
            User string
        }{User: user.(string), Error: ""}
		tmpl.Execute(w, data)
	}
}

func del_support(w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open("sqlite3", "my_db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

	_, er := db.Exec("DELETE FROM support WHERE id = ?", r.FormValue("support_id"))
	if er != nil {
		http.Error(w, er.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/support", http.StatusSeeOther)
}

func feedback(w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open("sqlite3", "my_db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
	session, err := store.Get(r, "user-session")
	user, ok := session.Values["user"]
	rows, err := db.Query("SELECT id, name, email, mes FROM feedback")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var feed []FeedbackItem

	for rows.Next() {
		var n FeedbackItem
		if err := rows.Scan(&n.Id, &n.Name, &n.Email, &n.Text); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		feed = append(feed, n)
	}
	tmpl, _ := template.ParseFiles("templates/feedback.html")
	data := struct{
		Id int
		Error string 
		User string
		Feedback []FeedbackItem
	}{Error: "", Feedback: feed}
	if ok {
		data.User = user.(string)
	} else{
		data.User = ""
	}
	tmpl.Execute(w, data)
}

func del_feedback(w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open("sqlite3", "my_db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

	_, er := db.Exec("DELETE FROM feedback WHERE id = ?", r.FormValue("feedback_id"))
	if er != nil {
		http.Error(w, er.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/feedback", http.StatusSeeOther)
}

func rent(w http.ResponseWriter, r *http.Request) {
    db, err := sql.Open("sqlite3", "my_db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    session, err := store.Get(r, "user-session")
    user, ok := session.Values["user"]
	var rent []RentItem
	nameFilter := r.FormValue("name-f")
	if nameFilter != ""{
		query := "SELECT id, name, square, price, image, phone, email FROM rent WHERE 1=1" // Начинаем с WHERE 1=1
		// Фильтр по названию (если введено ключевое слово)
		if nameFilter != "" {
			query += " AND name LIKE ?"
		}

		// Выполните SQL-запрос
		rows, err := db.Query(query, "%"+nameFilter+"%")
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()

		for rows.Next() {
			var n RentItem
			if err := rows.Scan(&n.Id, &n.Name, &n.Square, &n.Price, &n.Image, &n.Phone, &n.Email); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			n.Image = strings.Split(n.Image, " ")[0]
			db.QueryRow("SELECT id FROM basket WHERE user_id = ? AND rent_id = ?", session.Values["id"], n.Id).Scan(&n.InBasket)
			rent = append(rent, n)
    }
	}else{
		priceFilter := r.FormValue("price-f")
		squareFilter := r.FormValue("square-f")
	
		// Измените SQL-запрос в соответствии с фильтрами
		query := "SELECT id, name, square, price, image, phone, email FROM rent WHERE 1=1" // Начинаем с WHERE 1=1
	
		// Фильтр по цене
		
		if priceFilter == "cheapest" {
			query += " ORDER BY price ASC" // Сначала дешевые
		} else if priceFilter == "expensive" {
			query += " ORDER BY price DESC" // Сначала дорогие
		}
	
		// Фильтр по площади
		if squareFilter == "small" {
			query += " ORDER BY square ASC" // Сначала маленькие
		} else if squareFilter == "large" {
			query += " ORDER BY square DESC" // Сначала большие
		}
	
		// Выполните SQL-запрос
		rows, err := db.Query(query)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer rows.Close()
	
		for rows.Next() {
			var n RentItem
			if err := rows.Scan(&n.Id, &n.Name, &n.Square, &n.Price, &n.Image, &n.Phone, &n.Email); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
			n.Image = strings.Split(n.Image, " ")[0]
			db.QueryRow("SELECT id FROM basket WHERE user_id = ? AND rent_id = ?", session.Values["id"], n.Id).Scan(&n.InBasket)
			rent = append(rent, n)
		}
	}
    tmpl, _ := template.ParseFiles("templates/rent.html")
    data := struct {
        Id    int
        Error string
        User  string
        Rent  []RentItem
    }{Error: "", Rent: rent}
    if ok {
        data.User = user.(string)
    } else {
        data.User = ""
    }
    tmpl.Execute(w, data)
}

func del_rent(w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open("sqlite3", "my_db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

	_, er := db.Exec("DELETE FROM rent WHERE id = ?", r.FormValue("rent_id"))
	if er != nil {
		http.Error(w, er.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, "/rent", http.StatusSeeOther)
}

func add_rent(w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open("sqlite3", "my_db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()
	session, err := store.Get(r, "user-session")
	user, ok := session.Values["user"]
	if !ok || user != "admin"{
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}else if r.Method == http.MethodPost {
		r.ParseForm()
		images := r.Form["images[]"]
		image := ""
		for ind, file := range images{
			if ind == 0{
				image += "../static/images/" + file
			}else{
				image += " ../static/images/" + file
			}	
		}
		name := r.FormValue("name")
		square := r.FormValue("square")
		price := r.FormValue("price")
		phone := r.FormValue("phone")
		email := r.FormValue("email")
		about := r.FormValue("about")
		_, err = db.Exec("INSERT INTO rent (image, name, square, price, phone, email, about) VALUES (?, ?, ?, ?, ?, ?, ?)", image, name, square, price, phone, email, about)
		if err != nil {
			log.Fatal(err)
			http.Error(w, "Ошибка при сохранении данных в базу данных", http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, "/account", http.StatusSeeOther)
	}else{
		tmpl, _ := template.ParseFiles("templates/add_rent.html")
		data := struct{ 
            Error string 
            User string
        }{User: user.(string), Error: ""}
		tmpl.Execute(w, data)
	}
}

func basket(w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open("sqlite3", "my_db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    session, err := store.Get(r, "user-session")
    user, ok := session.Values["user"]
	var rent []RentItem
	rows, err := db.Query("SELECT rent.* FROM basket JOIN rent ON basket.rent_id = rent.id WHERE basket.user_id = ?", session.Values["id"])
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	id_temp := 1
	cost := 0
	for rows.Next() {
		var n RentItem
		if err := rows.Scan(&n.Id, &n.Name, &n.Square, &n.Price, &n.Image, &n.Phone, &n.Email, &n.About); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		cost += n.Price
		n.Ind = id_temp
		id_temp += 1
		rent = append(rent, n)
	}
    tmpl, _ := template.ParseFiles("templates/basket.html")
    data := struct {
        Id    int
        Error string
        User  string
        Rent  []RentItem
		Cost int
    }{Error: "", Rent: rent, Cost: cost}
    if ok {
        data.User = user.(string)
    } else {
        data.User = ""
    }
    tmpl.Execute(w, data)
}

func add_basket(w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open("sqlite3", "my_db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

	session, _ := store.Get(r, "user-session")
	user_id, _ := session.Values["id"]
	id_test := ""
	db.QueryRow("SELECT id FROM basket WHERE user_id = ? AND rent_id = ?", user_id, r.FormValue("rent_id")).Scan(&id_test)
	if id_test != ""{
		http.Redirect(w, r, "/rent", http.StatusSeeOther)
	}else{
		_, er := db.Exec("INSERT INTO basket (user_id, rent_id) VALUES (?, ?)", user_id, r.FormValue("rent_id"))

		if er != nil {
			http.Error(w, er.Error(), http.StatusInternalServerError)
			return
		}
		http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
	}
}

func del_basket(w http.ResponseWriter, r *http.Request) {
	db, err := sql.Open("sqlite3", "my_db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

	session, _ := store.Get(r, "user-session")
	user_id, _ := session.Values["id"]

	_, er := db.Exec("DELETE FROM basket WHERE user_id = ? AND rent_id = ?", user_id, r.FormValue("basket_id"))
	if er != nil {
		http.Error(w, er.Error(), http.StatusInternalServerError)
		return
	}
	http.Redirect(w, r, r.Header.Get("Referer"), http.StatusSeeOther)
}

func rent_info(w http.ResponseWriter, r *http.Request) {
	id_temp := r.URL.Query().Get("param")

	db, err := sql.Open("sqlite3", "my_db")
    if err != nil {
        log.Fatal(err)
    }
    defer db.Close()

    session, err := store.Get(r, "user-session")
    user, ok := session.Values["user"]
	
	var rent RentItem
	er := db.QueryRow("SELECT * FROM rent WHERE id = ?", id_temp).Scan(&rent.Id, &rent.Name, &rent.Square, &rent.Price, &rent.Image, &rent.Phone, &rent.Email, &rent.About)
	if er != nil {
		http.Redirect(w, r, "/", http.StatusSeeOther)
	}
	db.QueryRow("SELECT id FROM basket WHERE user_id = ? AND rent_id = ?", session.Values["id"], rent.Id).Scan(&rent.InBasket)

    tmpl, _ := template.ParseFiles("templates/rent_info.html")
    data := struct {
        Error string
        User  string
        Rent  RentItem
    }{Error: "", Rent: rent}
    if ok {
        data.User = user.(string)
    } else {
        data.User = ""
    }
    tmpl.Execute(w, data)
}

func handle_request() {
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	http.HandleFunc("/", index)
	http.HandleFunc("/news", news)
	http.HandleFunc("/card", card)
	http.HandleFunc("/contacts", contacts)
	http.HandleFunc("/registration", registr)
	http.HandleFunc("/sale", sale)
	http.HandleFunc("/support", support)
    http.HandleFunc("/login", login)
	http.HandleFunc("/logout", logout)
	http.HandleFunc("/add_news", add_news)
	http.HandleFunc("/del_news", del_news)
	http.HandleFunc("/account", account)
	http.HandleFunc("/add_card", add_card)
	http.HandleFunc("/add_sale", add_sale)
	http.HandleFunc("/del_sale", del_sale)
	http.HandleFunc("/add_support", add_support)
	http.HandleFunc("/del_support", del_support)
	http.HandleFunc("/feedback", feedback)
	http.HandleFunc("/del_feedback", del_feedback)
	http.HandleFunc("/rent", rent)
	http.HandleFunc("/add_rent", add_rent)
	http.HandleFunc("/del_rent", del_rent)
	http.HandleFunc("/basket", basket)
	http.HandleFunc("/add_basket", add_basket)
	http.HandleFunc("/rent_info", rent_info)
	http.HandleFunc("/del_basket", del_basket)
	http.ListenAndServe(":8080", nil)
}

func main() {
	handle_request()
}

