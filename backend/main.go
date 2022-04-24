package main

import (
	"database/sql"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type order_smena struct {
	Num   int     // номер заказа
	Price float64 // сумма заказа по приложению
	Tea   float64 // сумма чаевых
	Typ   string  // тип заказа n, t, o, k
}

type order_smena_text struct {
	Num   string // номер заказа
	Order string // заказ преобразованный в строку для вывода в шаблон
	Typ   string
}

type close_smena struct {
	Date string // дата
	Ok   string // для ошибочной даты
}

type kmh_text struct {
	// тип для передачи в шаблон kmh
	Num   string // номер смены
	Date  string // дата смены
	Km    string // пробег за смену
	H     string // время смены
	Price string // касса за смену
	Tea   string // чаевые за смену
	Count string // кол-во заказов за смену
	Prof  string // прибыль за смену
	Eff   string // эффективность км/ч
}

type rep struct {
	// структура для передачи данных в шаблон report
	Month       string   // отчетный месяц
	NalSum      string   // сумма налички за месяц
	NalCount    string   // количество наличных заказов за месяц
	Nal         []string // срез с наличными заказами
	TermSum     string   // сумма терминалов за месяц
	TermCount   string   // количество терминальных заказов за месяц
	Term        []string // соез с терминальными заказами
	OnlineSum   string   // сумма онлайнов за месяц
	OnlineCount string   // количество онлайн заказов за месяц
	Online      []string // срез с онлайн заказами
	KampSum     string   // сумма компаний за месяц
	KampCount   string   // количество заказов компания за месяц
	Kamp        []string // срез заказов компания
	TotalSum    string   // общая сумма всех заказов
	TotalCount  string   // общее количество всех заказов

	ComDis    string // комиссия диспетчера
	ComPer    string // комиссия перевозчика
	ComSum    string // ComDis + ComPer
	PayTerm   string // оплачено терминалами
	PayOnline string // оплачено онлайнами
	PaySum    string // PayTerm + PayOnline
	Balance   string // ComSum - PaySum

	Coment string // для коментария
}

type smen struct {
	// структура для передачи данных в шаблон smena
	Date         string             // дата открытой смены
	NalSum       string             // сумма налички за смену
	NalCount     string             // количество наличных заказов за смену
	TermSum      string             // сумма терминалов за смену
	TermCount    string             // количество терминальных заказов за смену
	NalTermCount string             // количество нал. + терм. заказов
	OnlineSum    string             // сумма онлайнов за смену
	OnlineCount  string             // количество онлайн заказов за смену
	KampSum      string             // сумма компаний за смену
	KampCount    string             // количество заказов компания за смену
	TotalSum     string             // общая сумма всех заказов
	TotalCount   string             // общее количество всех заказов
	Order        []order_smena_text // срез для заказов
	Coment       string             // для коментария
}

var ComDis float64       // комиссия диспетчера (в %)
var ComPer int           // комиссия перевозчика за свои услуги (в рублях)
var ComPerTer float64    // комиссия перевозчика за обналичку терминалов (в %)
var ComPerOnline float64 // комиссия перевозчика за обналичку онлайнов (в %)
var FuelCons float64     // расход топлива (л/100км)
var FuelPrice float64    // стоимость топлива (в рублях)
var WorkDay int          // количество рабочих дней - для расчета комисси за смену

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func createTablesDB() {
	// создание таблицы и файла БД, если их нет

	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	if err != nil {
		fmt.Println("ошибка открытия(создания) файла taxi.db")
		log.Fatal(err)
	}
	defer db.Close()
	// таблица для заказов на смене
	// номер заказа, сумма, чаевые, тип заказа (n, t, o, k)
	table := `CREATE TABLE IF NOT EXISTS smena (
 num INTEGER PRIMARY KEY AUTOINCREMENT,
 price NUMERIC(3,2),
 tea NUMERIC(3,2),
 typ CHAR
 );`
	mySqlExec(db, table)

	// таблица хранения всех заказов (обновляется при закрытии смены)
	// номер записи, дата заказа, сумма, чаевые, тип заказа (n, t, o, k)
	table = `CREATE TABLE IF NOT EXISTS orders (
 orders_id INTEGER PRIMARY KEY AUTOINCREMENT,
 date TEXT, 
 price NUMERIC(3,2),
 tea NUMERIC(3,2),
 typ CHAR);`
	mySqlExec(db, table)

	// таблица с данными о времени и пробеге по сменам
	// номер записи, дата открытия смены, пробег, время
	table = `CREATE TABLE IF NOT EXISTS kmh (
 kmh_id INTEGER PRIMARY KEY AUTOINCREMENT,
 date TEXT, 
 km INT,
 h NUMERIC(2,1) );`
	mySqlExec(db, table)

	// таблица для хранения основных переменных
	// date - дата открытой смены
	// comdis - комиссия диспетчера (в %)
	// comper - комиссия перевозчика за свои услуги (в рублях)
	// comperter - комиссия перевозчика за обналичку терминалов (в %)
	// comperonline - комиссия перевозчика за обналичку онлайнов (в %)
	// fuelcons - расход топлива (л/100км)
	// fuelprice - стоимость топлива (в рублях)
	// workday - количество рабочих дней - для расчета комисси за смену
	table = `CREATE TABLE IF NOT EXISTS setings (
 seting_id INTEGER PRIMARY KEY AUTOINCREMENT,
 date TEXT DEFAULT "",
 comdis NUMERIC(2,2),
 comper INTEGER,
 comperter NUMERIC(2,2),
 comperonline NUMERIC(2,2),
 fuelcons NUMERIC(2,2),
 fuelprice NUMERIC(2,2),
 workday);`
	mySqlExec(db, table)
	// проверяем, есть ли в таблице set данные в поле date
	record, err := db.Query("SELECT date FROM setings")
	if err != nil {
		log.Fatal(err)
	}
	defer record.Close()
	date := ""
	for record.Next() {
		record.Scan(&date)
		if err != nil {
			fmt.Println("Ошибка record.Scan")
			panic(err)
		}
	}
	if date == "" {
		//если нет данных в поле date, значит таблица только создана и в ней нет данных, добавляем их
		records := `INSERT INTO setings(date, comdis, comper, comperter, comperonline, fuelcons,
                    fuelprice, workday) VALUES ("close", 20, 170, 3, 3, 12, 1.24, 24)`
		// "close" - значит смена закрыта смена
		query, err := db.Prepare(records)
		if err != nil {
			log.Fatal(err)
		}
		_, err = query.Exec()
		if err != nil {
			log.Fatal(err)
		}
	}
}

func loadSettingsDB() {
	// читает основные переменные из БД
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	record, err := db.Query("SELECT comdis, comper, comperter, comperonline, fuelcons," +
		" fuelprice, workday FROM setings")
	if err != nil {
		log.Fatal(err)
	}
	defer record.Close()
	for record.Next() {
		record.Scan(&ComDis, &ComPer, &ComPerTer, &ComPerOnline, &FuelCons, &FuelPrice, &WorkDay)
		if err != nil {
			fmt.Println("Ошибка record.Scan")
			panic(err)
		}
	}
}

func mySqlExec(db *sql.DB, s string) {
	query, err := db.Prepare(s)
	if err != nil {
		fmt.Println("ошибка создания запроса методом Prepare")
		log.Fatal(err)
	}
	_, err = query.Exec()
	if err != nil {
		fmt.Println("ошибка выполнения запроса методом Exec")
		log.Fatal(err)
	}
	query.Close()
}

func smenaDB() []order_smena_text {
	// возвращает срез заказов за смену
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	record, err := db.Query("SELECT * FROM smena")
	if err != nil {
		log.Fatal(err)
	}
	defer record.Close()
	orders := []order_smena_text{}
	order := order_smena{}
	orderText := order_smena_text{}
	for record.Next() {
		record.Scan(&order.Num, &order.Price, &order.Tea, &order.Typ)
		if err != nil {
			fmt.Println("Ошибка record.Scan")
			panic(err)
		}

		orderText.Num = fmt.Sprintf(" %d ", order.Num)
		orderText.Order = fmt.Sprintf(" %.2f    (%.2f)", order.Price, order.Tea)
		orderText.Typ = order.Typ
		orders = append(orders, orderText)
	}
	return orders
}

func kmhDB(fuel float64, fuelPrice float64, comDis float64, comPer float64) []kmh_text {
	// принимает расход топлива в л/100км (10.5) fuel
	// 			стоимость топлива в руб (1.2) fuelPrice
	// 			комиссию дисп в % (20) comDis
	//			комиссию перевозчика в руб за смену (7.7) comPer
	// возвращает срез смен из таблицы kmh
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	record, err := db.Query("SELECT kmh_id, date, km , h, price, tea, count FROM kmh INNER JOIN" +
		" (SELECT date AS orders_date, sum(price) AS price, sum(tea) AS tea, count() AS count FROM orders" +
		" GROUP BY date) ON date = orders_date ORDER BY date")
	if err != nil {
		log.Fatal(err)
	}
	defer record.Close()

	smens := []kmh_text{}
	var num, km, count int
	var date string
	var h, price, tea float64
	smen := kmh_text{}
	for record.Next() {
		record.Scan(&num, &date, &km, &h, &price, &tea, &count)
		if err != nil {
			fmt.Println("Ошибка record.Scan")
			panic(err)
		}

		smen.Num = fmt.Sprintf("%d", num)
		smen.Date = fmt.Sprintf("%s", date)
		smen.Km = fmt.Sprintf("%d", km)
		smen.H = fmt.Sprintf("%.1f", h)
		smen.Price = fmt.Sprintf("%.0f", price)
		smen.Tea = fmt.Sprintf("%.0f", tea)
		smen.Count = fmt.Sprintf("%d", count)
		prof := price - (fuel / 100 * float64(km) * fuelPrice) - (price / 100 * comDis) - comPer + tea
		smen.Prof = fmt.Sprintf("%.0f", prof)
		eff := prof / h
		smen.Eff = fmt.Sprintf("%.1f", eff)
		smens = append(smens, smen)
	}
	return smens
}

func indexNumDB() {
	// нумерует заказы в таблице smena по порядку
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// читаем в срез nums все записи поля num (номера заказа)
	record, err := db.Query("SELECT num FROM smena")
	if err != nil {
		log.Fatal(err)
	}
	defer record.Close()
	nums := []int{}
	num := 0
	for record.Next() {
		record.Scan(&num)
		if err != nil {
			fmt.Println("Ошибка record.Scan")
			panic(err)
		}
		nums = append(nums, num)
	}

	// переиндексируем в таблице smena поле num по порядку
	records := ``
	for i := 1; i <= len(nums); i++ {
		records = `UPDATE smena SET num = ? WHERE num = ?`
		query, err := db.Prepare(records)
		if err != nil {
			log.Fatal(err)
		}
		_, err = query.Exec(i, nums[i-1])
		if err != nil {
			log.Fatal(err)
		}
	}
}

func indexNumSmenDB() {
	// нумерует смены в таблице kmh по порядку
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// читаем в срез nums все записи поля kmh_id (номера смен)
	record, err := db.Query("SELECT kmh_id FROM kmh")
	if err != nil {
		log.Fatal(err)
	}
	defer record.Close()
	nums := []int{}
	num := 0
	for record.Next() {
		record.Scan(&num)
		if err != nil {
			fmt.Println("Ошибка record.Scan")
			panic(err)
		}
		nums = append(nums, num)
	}

	// переиндексируем в таблице smena поле num по порядку
	records := ``
	for i := 1; i <= len(nums); i++ {
		records = `UPDATE kmh SET kmh_id = ? WHERE kmh_id = ?`
		query, err := db.Prepare(records)
		if err != nil {
			log.Fatal(err)
		}
		_, err = query.Exec(i, nums[i-1])
		if err != nil {
			log.Fatal(err)
		}
	}
}

func smenaSumTypDB(typ string) (string, string) {
	// возвращает сумму заказов за смену по типу  в текстовом виде
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	record, err := db.Query("SELECT sum(price), count() FROM smena WHERE typ = ?", typ)
	if err != nil {
		log.Fatal(err)
	}
	defer record.Close()
	var i float64 // для суммы всех заказов типа typ
	var c int     // для количества всех заказов типа typ
	var s, s1 string
	for record.Next() {
		record.Scan(&i, &c)
		if err != nil {
			fmt.Println("Ошибка record.Scan")
			panic(err)
		}

		s = fmt.Sprintf("%.2f", i)
		s1 = fmt.Sprintf("%d", c)
	}
	return s, s1
}

func smenaSumDB() (string, string) {
	// возвращает сумму заказов за смену в текстовом виде касса + чай = итог
	// а вторым аргументом общее количество заказов за смену
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	record, err := db.Query("SELECT sum(price), sum(tea), count() FROM smena")
	if err != nil {
		log.Fatal(err)
	}
	defer record.Close()
	var price, tea float64
	var count int
	var s, c string
	for record.Next() {
		record.Scan(&price, &tea, &count)
		if err != nil {
			fmt.Println("Ошибка record.Scan")
			panic(err)
		}
		c = fmt.Sprintf("%d", count)
		s = fmt.Sprintf("касса %.2f  +  чай %.2f  =  %.2f", price, tea, price+tea)
	}
	return s, c
}

func addOrderDB(price string, tea string, typ string) {
	// добавляет заказ в БД в таблице smena
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	records := `INSERT INTO smena(price, tea, typ) VALUES (?, ?, ?)`
	query, err := db.Prepare(records)
	if err != nil {
		log.Fatal(err)
	}
	_, err = query.Exec(price, tea, typ)
	if err != nil {
		log.Fatal(err)
	}
	indexNumDB()
}

func editOrderDB(num string, price string, tea string, typ string) {
	// изменяет заказ номер num в БД в таблице smena
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	records := `UPDATE smena SET price = ?, tea = ?, typ = ? WHERE num = ?`
	query, err := db.Prepare(records)
	if err != nil {
		log.Fatal(err)
	}
	_, err = query.Exec(price, tea, typ, num)
	if err != nil {
		log.Fatal(err)
	}
}

func delOrderDB(n int) {
	// удаление заказа n
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	records := `DELETE FROM smena WHERE num = ?`
	query, err := db.Prepare(records)
	if err != nil {
		log.Fatal(err)
	}
	_, err = query.Exec(n)
	if err != nil {
		log.Fatal(err)
	}

	indexNumDB()
}

func dateFromSetingsDB() string {
	// возвращает дату открытой смены из таблицы setings
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	record, err := db.Query("SELECT date FROM setings")
	if err != nil {
		log.Fatal(err)
	}
	defer record.Close()
	date := ""
	for record.Next() {
		record.Scan(&date)
		if err != nil {
			fmt.Println("Ошибка record.Scan")
			panic(err)
		}
	}
	return date
}

func dateToSetingsDB(date string) {
	//изменяет в таблице setings поле date на date ("close" значит, что сейчас смена закрыта)
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	records := `UPDATE setings SET date = ?`
	query, err := db.Prepare(records)
	if err != nil {
		log.Fatal(err)
	}
	_, err = query.Exec(date)
	if err != nil {
		log.Fatal(err)
	}
}

func addKmhDB(date string, km int, h float64) {
	// добавляет итоги смены в таблицу kmh
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	records := `INSERT INTO kmh(date, km, h) VALUES (?, ?, ?)`
	query, err := db.Prepare(records)
	if err != nil {
		log.Fatal(err)
	}
	_, err = query.Exec(date, km, h)
	if err != nil {
		log.Fatal(err)
	}
	indexNumSmenDB()
}

func smenaTOordersDB(date string) {
	// переносит данные из таблицы smena в orders
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	records := `INSERT INTO orders (date, price, tea, typ) SELECT ?, price, tea, typ FROM smena`
	query, err := db.Prepare(records)
	if err != nil {
		log.Fatal(err)
	}
	_, err = query.Exec(date)
	if err != nil {
		log.Fatal(err)
	}
	// очищает таблицу smena
	records = `DELETE FROM smena`
	query, err = db.Prepare(records)
	if err != nil {
		log.Fatal(err)
	}
	_, err = query.Exec()
	if err != nil {
		log.Fatal(err)
	}

}

func reportDB(month string, typ string) (float64, int, []string) {
	// возвращает сумму заказов за месяц month по типу typ в текстовом виде в sum
	// количество в cont и список заказов в orders
	var record *sql.Rows
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()
	monthSQL := fmt.Sprintf("__.%s.22", month)
	if typ == "all" {
		record, err = db.Query("SELECT sum(price), count() FROM orders")
		if err != nil {
			log.Fatal(err)
		}
	} else {
		record, err = db.Query("SELECT sum(price), count() FROM orders WHERE typ = ? AND date LIKE ?", typ, monthSQL)
		if err != nil {
			log.Fatal(err)
		}
	}
	defer record.Close()
	var sum float64 // для суммы всех заказов типа typ
	var count int   // для количества всех заказов типа typ
	for record.Next() {
		record.Scan(&sum, &count)
		if err != nil {
			fmt.Println("Ошибка record.Scan")
			panic(err)
		}
	}

	record, err = db.Query("SELECT date, price, tea FROM orders WHERE typ = ? AND date LIKE ? ORDER BY date", typ, monthSQL)
	if err != nil {
		log.Fatal(err)
	}
	var date, s string
	var price, tea float64
	var n int
	var orders []string
	for record.Next() {
		record.Scan(&date, &price, &tea)
		if err != nil {
			fmt.Println("Ошибка record.Scan")
			panic(err)
		}
		n++
		s = fmt.Sprintf("%d.  %s  %.2fр %.2fр", n, date, price, tea)
		orders = append(orders, s)
	}

	return sum, count, orders
}

func delSmenDB(n int) {
	// удаление смены n из БД
	var date string
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	// читаем дату смены n
	record, err := db.Query("SELECT date FROM kmh WHERE kmh_id = ?", n)
	if err != nil {
		log.Fatal(err)
	}
	defer record.Close()
	for record.Next() {
		record.Scan(&date)
		if err != nil {
			fmt.Println("Ошибка record.Scan")
			panic(err)
		}
	}

	// удаляем смену n из таблицы kmh
	records := `DELETE FROM kmh WHERE kmh_id = ?`
	query, err := db.Prepare(records)
	if err != nil {
		log.Fatal(err)
	}
	_, err = query.Exec(n)
	if err != nil {
		log.Fatal(err)
	}

	// удаляем заказы с датой из таблицы orders
	records = `DELETE FROM orders WHERE date = ?`
	query, err = db.Prepare(records)
	if err != nil {
		log.Fatal(err)
	}
	_, err = query.Exec(date)
	if err != nil {
		log.Fatal(err)
	}
	indexNumSmenDB()
}

func editSmenDB(num string, km string, h string) {
	// изменяет смену номер num в БД в таблице kmh
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	records := `UPDATE kmh SET km = ?, h = ? WHERE kmh_id = ?`
	query, err := db.Prepare(records)
	if err != nil {
		log.Fatal(err)
	}
	_, err = query.Exec(km, h, num)
	if err != nil {
		log.Fatal(err)
	}
}

func okOrder(in string) []string {
	// из строки введенной в форме выдает правильный срез для добавления заказа
	// в котором inSl[0] - тип заказа
	// inSl[1] - стоимость
	// inSl[2] - чаевые
	inSl := strings.Split(in, " ")
	fmt.Println(inSl)
	if inSl[0] == "т" || inSl[0] == "t" {
		inSl[0] = "t"
	} else if inSl[0] == "о" || inSl[0] == "o" {
		inSl[0] = "o"
	} else if inSl[0] == "к" || inSl[0] == "k" {
		inSl[0] = "k"
	} else {
		_, err := strconv.ParseFloat(inSl[0], 64)
		if err == nil {
			//если первый элемент число (буква не была введена)
			if len(inSl) == 1 {
				// и если нет чаевых (введено только 1 число)
				inSl = append(inSl, inSl[0])
				inSl = append(inSl, "0")
				inSl[0] = "n"
			} else {
				// и есть чаевые (введено 2 числа)
				inSl = append(inSl, inSl[1])
				inSl[1] = inSl[0]
				inSl[0] = "n"
			}
		} else {
			// если первый элемент неизвестная буква
			inSl[0] = "n"
		}
	}
	if len(inSl) == 2 {
		// если нет чая
		inSl = append(inSl, "0")
	}
	if inSl[2] != "0" {
		// после версии 1.0.2 в качестве 2го числа при вводе заказа надо вводить не чаевые, а полную сумму оплаты
		// (заказ + чаевые)
		i2, err := strconv.ParseFloat(inSl[2], 64)
		if err != nil {
			fmt.Println("ошибка strconv.ParseFloat(inSl[2],64)")
			fmt.Println(err)
			panic(err)
		}
		i1, err := strconv.ParseFloat(inSl[1], 64)
		if err != nil {
			fmt.Println("ошибка strconv.ParseFloat(inSl[1],64)")
			fmt.Println(err)
			panic(err)
		}
		i := i2 - i1
		inSl[2] = fmt.Sprintf("%.2f", i)
	}
	return inSl
}

func smena(w http.ResponseWriter, r *http.Request) {
	//todo проверку куков (для авторизации)
	date := dateFromSetingsDB()
	if date == "close" {
		// если смена закрыта
		t, err := template.ParseFiles("./templates/closesmene.html",
			"./templates/header.html", "./templates/footer.html")
		if err != nil {
			fmt.Println("ошибка template.ParseFiles")
			fmt.Println(err.Error())
		}
		var outt close_smena
		now := time.Now()
		outt.Date = now.Format("02.01.06")
		outt.Ok = "ok"
		t.ExecuteTemplate(w, "closesmene", outt)
	} else {
		// если смена открыта
		t, err := template.ParseFiles("./templates/index.html",
			"./templates/header.html", "./templates/footer.html")
		if err != nil {
			fmt.Println("ошибка template.ParseFiles")
			fmt.Println(err.Error())
		}

		var out smen
		out.Date = date
		out.Order = smenaDB()
		out.NalSum, out.NalCount = smenaSumTypDB("n")
		out.TermSum, out.TermCount = smenaSumTypDB("t")
		out.OnlineSum, out.OnlineCount = smenaSumTypDB("o")
		out.KampSum, out.KampCount = smenaSumTypDB("k")
		out.TotalSum, out.TotalCount = smenaSumDB()
		out.Coment = "ok"

		t.ExecuteTemplate(w, "index", out)
	}
}

func openSmena(w http.ResponseWriter, r *http.Request) {
	var ok int
	date := r.FormValue("date")
	dateSl := strings.Split(date, ".")
	day, err := strconv.Atoi(dateSl[0])
	if err != nil {
		ok++
	}
	if day < 10 {
		dateSl[0] = "0" + strconv.Itoa(day)
	}
	if day > 31 {
		ok++
	}
	mes, err := strconv.Atoi(dateSl[1])
	if err != nil {
		ok++
	}
	if mes > 12 {
		ok++
	}
	_, err = strconv.Atoi(dateSl[2])
	if err != nil {
		ok++
	}
	if ok == 0 {
		date = strings.Join(dateSl, ".")
		fmt.Println("Открывается смена ", date)
		dateToSetingsDB(date)
		defer smena(w, r)
	} else {
		defer errorDate(w, r)
	}
}

func addorder(w http.ResponseWriter, r *http.Request) {
	in := r.FormValue("in")
	if in != "" {
		inSl := okOrder(in)
		addOrderDB(inSl[1], inSl[2], inSl[0])
	}
	defer smena(w, r)
}

func delOrder(w http.ResponseWriter, r *http.Request) {
	in := r.FormValue("in")
	i, _ := strconv.Atoi(in)
	delOrderDB(i)
	defer corect(w, r)
}

func editOrder(w http.ResponseWriter, r *http.Request) {
	// номер редактируемого заказа
	num := r.FormValue("num")
	// новые данные заказа
	edit := r.FormValue("edit")
	inSl := okOrder(edit)
	editOrderDB(num, inSl[1], inSl[2], inSl[0])

	defer corect(w, r)
}

func corect(w http.ResponseWriter, r *http.Request) {

	t, err := template.ParseFiles("./templates/corect.html",
		"./templates/header.html", "./templates/footer.html")
	if err != nil {
		fmt.Println("ошибка template.ParseFiles")
		fmt.Println(err.Error())
	}

	var out smen
	out.Order = smenaDB()
	out.NalSum, out.NalCount = smenaSumTypDB("n")
	out.TermSum, out.TermCount = smenaSumTypDB("t")
	out.OnlineSum, out.OnlineCount = smenaSumTypDB("o")
	out.KampSum, out.KampCount = smenaSumTypDB("k")
	out.TotalSum, out.TotalCount = smenaSumDB()
	out.Coment = "ok"

	t.ExecuteTemplate(w, "corect", out)

}

func sclose(w http.ResponseWriter, r *http.Request) {
	// закрывает смену
	// очищает таблицу smena
	// удаляет дату из таблицы set
	t, err := template.ParseFiles("./templates/close.html",
		"./templates/header.html", "./templates/footer.html")
	if err != nil {
		fmt.Println("ошибка template.ParseFiles")
		fmt.Println(err.Error())
	}

	var out smen
	out.Date = dateFromSetingsDB()
	out.NalSum, out.NalCount = smenaSumTypDB("n")
	out.TermSum, out.TermCount = smenaSumTypDB("t")
	out.OnlineSum, out.OnlineCount = smenaSumTypDB("o")
	out.KampSum, out.KampCount = smenaSumTypDB("k")
	out.TotalSum, out.TotalCount = smenaSumDB()
	out.Coment = "ok"
	count1, _ := strconv.Atoi(out.NalCount)
	count2, _ := strconv.Atoi(out.TermCount)
	count := count1 + count2
	out.NalTermCount = fmt.Sprintf("%d", count)

	t.ExecuteTemplate(w, "close", out)
}

func scloseForm(w http.ResponseWriter, r *http.Request) {
	kms := r.FormValue("km")
	km, _ := strconv.Atoi(kms)
	hs := r.FormValue("h")
	h, _ := strconv.ParseFloat(hs, 64)
	date := dateFromSetingsDB()
	addKmhDB(date, km, h) // добавляет итоги смены в таблицу kmh
	fmt.Println("Закрытие смены: ", km, "км, ", h, "ч")
	//closeDateKmhDB() // изменяет в таблице setings поле date на "close"
	dateToSetingsDB("close")
	smenaTOordersDB(date)
	defer smena(w, r)
}

func report(w http.ResponseWriter, r *http.Request) {
	var nalSum, termSum, onlineSum, kampSum, totalSum float64
	var nalCount, termCount, onlineCount, kampCount, totalCount int
	t, err := template.ParseFiles("./templates/report.html",
		"./templates/header.html", "./templates/footer.html")
	if err != nil {
		fmt.Println("ошибка template.ParseFiles")
		fmt.Println(err.Error())
	}
	out := rep{}
	out.Month = "04" //todo сделать ввод отчетного месяца
	nalSum, nalCount, out.Nal = reportDB(out.Month, "n")
	out.NalSum = fmt.Sprintf("%.2f", nalSum)
	out.NalCount = fmt.Sprintf("%d", nalCount)
	termSum, termCount, out.Term = reportDB(out.Month, "t")
	out.TermSum = fmt.Sprintf("%.2f", termSum)
	out.TermCount = fmt.Sprintf("%d", termCount)
	onlineSum, onlineCount, out.Online = reportDB(out.Month, "o")
	out.OnlineSum = fmt.Sprintf("%.2f", onlineSum)
	out.OnlineCount = fmt.Sprintf("%d", onlineCount)
	kampSum, kampCount, out.Kamp = reportDB(out.Month, "k")
	out.KampSum = fmt.Sprintf("%.2f", kampSum)
	out.KampCount = fmt.Sprintf("%d", kampCount)
	totalSum, totalCount, _ = reportDB(out.Month, "all")
	out.TotalSum = fmt.Sprintf("%.2f", totalSum)
	out.TotalCount = fmt.Sprintf("%d", totalCount)
	comDisR := (nalSum + termSum + onlineSum) / 100 * ComDis
	out.ComDis = fmt.Sprintf("%.2f", comDisR)
	out.ComPer = fmt.Sprintf("%d", ComPer)
	comSum := comDisR + float64(ComPer)
	out.ComSum = fmt.Sprintf("%.2f", comSum)
	payTerm := termSum / 100 * (100 - ComPerTer)
	out.PayTerm = fmt.Sprintf("%.2f", payTerm)
	payOnline := onlineSum / 100 * (100 - ComPerOnline)
	out.PayOnline = fmt.Sprintf("%.2f", payOnline)
	paySum := payTerm + payOnline
	out.PaySum = fmt.Sprintf("%.2f", paySum)
	out.Balance = fmt.Sprintf("%.2f", comSum-paySum)

	t.ExecuteTemplate(w, "report", out)
}

func kmh(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("./templates/kmh.html",
		"./templates/header.html", "./templates/footer.html")
	if err != nil {
		fmt.Println("ошибка template.ParseFiles")
		fmt.Println(err.Error())
	}

	out := kmhDB(FuelCons, FuelPrice, ComDis, float64(ComPer/WorkDay))

	t.ExecuteTemplate(w, "kmh", out)
}

func delSmen(w http.ResponseWriter, r *http.Request) {
	in := r.FormValue("in")
	i, _ := strconv.Atoi(in)
	delSmenDB(i)

	defer kmh(w, r)
}

func editSmen(w http.ResponseWriter, r *http.Request) {
	// номер редактируемой смены
	num := r.FormValue("num")
	//num, _ := strconv.Atoi(in)
	// новый километраж за смену
	km := r.FormValue("km")
	// новое время смены
	h := r.FormValue("h")
	editSmenDB(num, km, h)

	defer kmh(w, r)
}

func saveDb(w http.ResponseWriter, r *http.Request) {
	//todo сделать проверку пользователя для загрузки БД
	t, err := template.ParseFiles("./templates/corect.html",
		"./templates/header.html", "./templates/footer.html")
	if err != nil {
		fmt.Println("ошибка template.ParseFiles")
		fmt.Println(err.Error())
	}
	var out smen

	// parse data
	err = r.ParseMultipartForm(1024)
	check(err)

	// get 'file'
	file, handler, err := r.FormFile("file")
	check(err)
	if handler.Filename != "taxi.db" {
		// попытка загрузить не ту БД
		out.Coment = "errSaveDB"
	} else {
		oldDb, err := ioutil.ReadFile("../dir_db/taxi.db")
		check(err)
		err = ioutil.WriteFile("../dir_db/old_taxi.db", oldDb, 0644)
		check(err)
		fileName := "../dir_db/" + handler.Filename

		// read file bytes
		fileBytes, err := ioutil.ReadAll(file)
		check(err)

		// write bytes to a localfile
		err = ioutil.WriteFile(fileName, fileBytes, 0644)
		check(err)
		out.Coment = "okSaveDB"
		file.Close()
	}
	out.Order = smenaDB()
	out.NalSum, out.NalCount = smenaSumTypDB("n")
	out.TermSum, out.TermCount = smenaSumTypDB("t")
	out.OnlineSum, out.OnlineCount = smenaSumTypDB("o")
	out.KampSum, out.KampCount = smenaSumTypDB("k")
	out.TotalSum, out.TotalCount = smenaSumDB()

	t.ExecuteTemplate(w, "corect", out)
}

func errorDate(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("./templates/closesmene.html",
		"./templates/header.html", "./templates/footer.html")
	if err != nil {
		fmt.Println("ошибка template.ParseFiles")
		fmt.Println(err.Error())
	}
	var out close_smena
	now := time.Now()
	out.Date = now.Format("02.01.06")
	out.Ok = "err"
	t.ExecuteTemplate(w, "closesmene", out)
}

func main() {
	createTablesDB()
	loadSettingsDB()

	http.HandleFunc("/", smena)
	http.HandleFunc("/addorder", addorder)
	http.HandleFunc("/smena_close", sclose)
	http.HandleFunc("/smena_close_form", scloseForm)
	http.HandleFunc("/corect", corect)
	http.HandleFunc("/del_order", delOrder)
	http.HandleFunc("/edit_order", editOrder)
	http.HandleFunc("/open_smena", openSmena)
	http.HandleFunc("/report", report)
	http.HandleFunc("/kmh", kmh)
	http.HandleFunc("/del_smen", delSmen)
	http.HandleFunc("/edit_smen", editSmen)
	http.HandleFunc("/save_db", saveDb)

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	http.Handle("/dir_db/", http.StripPrefix("/dir_db/", http.FileServer(http.Dir("../dir_db/"))))
	log.Println(http.ListenAndServe(":5005", nil))
}
