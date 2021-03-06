package main

import (
	"crypto/md5"
	"database/sql"
	"encoding/hex"
	"fmt"
	_ "github.com/mattn/go-sqlite3"
	"html/template"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"os"
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

type Sett struct {
	UserName     string // имя пользователя
	FuelCons     string // расход топлива
	FuelPrice    string // цена топлива
	WorkDay      string // рабочих дней
	ComDis       string // комиссия диспетчера
	ComPer       string // комиссия перевозчика
	ComPerTer    string // комиссия перевозчика за терминалы
	ComPerOnline string // комиссия перевозчика за онлайны
	Coment       string // для коментария
}

type close_smena struct {
	UserName string // имя пользователя
	Date     string // дата
	Ok       string // для ошибочной даты
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
	UserName    string   // имя пользователя
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
	Distance    string   // общий пробег
	PayDist     string   // стоимость топлива за месяц
	Hours       string   // количество часов
	Count       string   // количество смен
	Salary      string   // Чистый заработок за вычетом топлива и комиссий
	SalaryH     string   // Salary / Hours

	ComDis    string // комиссия диспетчера
	ComPer    string // комиссия перевозчика
	ComSum    string // ComDis + ComPer
	PayTerm   string // оплачено терминалами
	PayOnline string // оплачено онлайнами
	PayKamp   string // оплачено компаниями
	PaySum    string // PayTerm + PayOnline
	Balance   string // ComSum - PaySum

	Coment string // для коментария
}

type smen struct {
	// структура для передачи данных в шаблон corect
	UserName     string             // имя пользователя
	SessionId    string             // id смены
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
	IndexComment string             // для комментария :)
}

type repair_form_type struct {
	// тип для заполнения формы редактирования ремонта
	Id     string
	Detail string
	Cod    string
	Date   string
	Km     string
}
type repairs_type struct {
	// под под структура для передачи данных в шаблон repair
	Repair_id string // id
	Date      string // дата замены
	Km        string // пробег при замене
	Cod       string // код детали
}
type details_type struct {
	// под структура для передачи данных в шаблон repair
	Detail  string         // деталь
	Date    string         // дата последней замены
	Passed  string         // пробег со времени последней замены
	Repairs []repairs_type // срез со случаями ремонтов
}
type repair_type struct {
	// структура для передачи данных в шаблон repair
	UserName string           // имя пользователя
	CurKm    string           // текущий пробег
	Details  []details_type   // срез для узлов
	Form     repair_form_type // тип для заполнения формы редактирования ремонта
}

var ComDis float64                     // комиссия диспетчера (в %)
var ComPer int                         // комиссия перевозчика за свои услуги (в рублях)
var ComPerTer float64                  // комиссия перевозчика за обналичку терминалов (в %)
var ComPerOnline float64               // комиссия перевозчика за обналичку онлайнов (в %)
var Month = map[string]string{}        // key - userid, value - отчетный месяц
var Km = map[string]string{}           // key - userid, value - текущий пробег
var IndexComment = map[string]string{} // key - userid, value - коментарий

func check(e error) {
	if e != nil {
		panic(e)
	}
}

func GetMd5(pas string) string {
	h := md5.New()
	h.Write([]byte(pas))
	return hex.EncodeToString(h.Sum(nil))
} //2

func versionDB(v int) int {
	// обновляет номер версии до значения полученного в v
	// если v=0, только возвращает версию БД, возвращает 0 если база старая и нет поля версии, -1 если taxi.db нет
	var ver int
	if _, err := os.Stat("../dir_db/taxi.db"); os.IsNotExist(err) {
		// если файл не найден
		return -1
	}

	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	if err != nil {
		fmt.Println("ошибка1 ", err)
	}
	defer db.Close()

	if v > 0 {
		records := `UPDATE settings SET ver = ?`
		query, err := db.Prepare(records)
		check(err)
		_, err = query.Exec(v)
		check(err)
		return v
	}

	record, err := db.Query("SELECT ver FROM settings")
	if err != nil {
		return 0
	}
	defer record.Close()
	for record.Next() {
		record.Scan(&ver)
	}
	return ver
} //2

func transformDBto2() {
	// трансформируем БД во 2 версию
	oldDb, err := ioutil.ReadFile("../dir_db/taxi.db")
	check(err)
	err = ioutil.WriteFile("../dir_db/taxi_ver1.db", oldDb, 0644)
	check(err)
	err = os.Remove("../dir_db/taxi.db")
	check(err)
	createTablesDB()

	dbOld, err := sql.Open("sqlite3", "../dir_db/taxi_ver1.db")
	check(err)
	defer dbOld.Close()

	// переносим данные из старой базы таблицы orders в новую базу в orders
	record, err := dbOld.Query("SELECT kmh_id, price, tea, typ FROM kmh INNER JOIN orders" +
		" ON kmh.date = orders.date ORDER BY orders.date;")
	check(err)
	defer record.Close()
	userId := "1"
	type orders struct {
		sessionId int
		price     float64
		tea       float64
		typ       string
	}
	temps := []orders{}
	var temp orders
	for record.Next() {
		record.Scan(&temp.sessionId, &temp.price, &temp.tea, &temp.typ)
		temps = append(temps, temp)
	}

	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	check(err)
	defer db.Close()
	records := `INSERT INTO orders(user_id, session_id, price, tea, type) VALUES (?, ?, ?, ?, ?)`
	query, err := db.Prepare(records)
	check(err)

	for i, t := range temps {
		fmt.Println("taxi.db/orders/string", i)
		_, err = query.Exec(userId, t.sessionId, t.price, t.tea, t.typ)
		check(err)
	}

	// переносим данные из старой базы таблицы kmh в новую базу в sessions
	record, err = dbOld.Query("SELECT kmh_id, date, km, h FROM kmh")
	check(err)
	type kmh struct {
		kmhId int
		date  string
		km    int
		h     float64
	}
	temps2 := []kmh{}
	var temp2 kmh
	for record.Next() {
		record.Scan(&temp2.kmhId, &temp2.date, &temp2.km, &temp2.h)
		temps2 = append(temps2, temp2)
	}
	records = `INSERT INTO sessions(session_id, date, user_id, km, h) VALUES (?, ?, ?, ?, ?)`
	query, err = db.Prepare(records)
	check(err)

	for i, t := range temps2 {
		fmt.Println("taxi.db/sessions/string", i)
		_, err = query.Exec(t.kmhId, t.date, userId, t.km, t.h)
		check(err)
	}

	// создаем данные в новой базе в таблице users
	hashPas := GetMd5("2666")
	records = `INSERT INTO users(name, password, session_id, date, fuelcons, fuelprice, workday)
 VALUES ("Rick", ?, 0, "close", 12, 1.24, 24)`
	query, err = db.Prepare(records)
	check(err)
	fmt.Println("taxi.db/userss/string 1")
	_, err = query.Exec(hashPas)
	check(err)

	// записываем в новой базе в таблице settings последний номер смены
	record, err = db.Query("SELECT max(session_id) FROM sessions")
	check(err)
	var maxsession int
	for record.Next() {
		record.Scan(&maxsession)
	}

	records = `UPDATE settings SET lastsession = ?`
	query, err = db.Prepare(records)
	check(err)
	_, err = query.Exec(maxsession)
	check(err)
} //2

func setSettingsDB(in Sett) {
	// вносит изменения в БД, в настройки пользователя и комиссии
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	check(err)
	defer db.Close()

	records := `UPDATE settings SET comdis = ?, comper = ?, comperter = ?, comperonline = ?`
	query, err := db.Prepare(records)
	check(err)
	_, err = query.Exec(in.ComDis, in.ComPer, in.ComPerTer, in.ComPerOnline)
	check(err)

	records = `UPDATE users SET fuelcons = ?, fuelprice = ?, workday = ? WHERE users_id = ?`
	query, err = db.Prepare(records)
	check(err)
	_, err = query.Exec(in.FuelCons, in.FuelPrice, in.WorkDay, in.UserName) // здесь в in.UserName - userid
	check(err)
}

func createTablesDB() {
	// создание таблицы и файла БД, если их нет

	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	if err != nil {
		fmt.Println("ошибка открытия(создания) файла taxi.db")
		panic(err)
	}
	defer db.Close()
	// таблица для заказов на смене
	// номер заказа, сумма, чаевые, тип заказа (n, t, o, k)
	table := `CREATE TABLE IF NOT EXISTS orderssession (
 os_id INTEGER PRIMARY KEY AUTOINCREMENT,
 user_id INTEGER,
 price NUMERIC(3,2),
 tea NUMERIC(3,2),
 type CHAR);`
	mySqlExec(db, table)

	// таблица хранения всех заказов (обновляется при закрытии смены)
	// номер записи, дата заказа, сумма, чаевые, тип заказа (n, t, o, k)
	table = `CREATE TABLE IF NOT EXISTS orders (
 orders_id INTEGER PRIMARY KEY AUTOINCREMENT,
 user_id INTEGER,
 session_id INTEGER,
 price NUMERIC(3,2),
 tea NUMERIC(3,2),
 type CHAR);`
	mySqlExec(db, table)

	// таблица с данными о времени и пробеге по сменам
	// номер записи, дата открытия смены, пробег, время
	table = `CREATE TABLE IF NOT EXISTS sessions (
 session_id INTEGER PRIMARY KEY AUTOINCREMENT,
 date WARCHAR(8),
 user_id INTEGER,
 km INTEGER,
 h NUMERIC(2,1));`
	mySqlExec(db, table)

	// таблица для хранения основных переменных
	// ver - версия БД
	// comdis - комиссия диспетчера (в %)
	// comper - комиссия перевозчика за свои услуги (в рублях)
	// comperter - комиссия перевозчика за обналичку терминалов (в %)
	// comperonline - комиссия перевозчика за обналичку онлайнов (в %)
	table = `CREATE TABLE IF NOT EXISTS settings (
 ver INTEGER,
 comdis NUMERIC(2,1),
 comper NUMERIC(2,2),
 comperter NUMERIC(2,1),
 comperonline NUMERIC(2,1),
 lastsession INTEGER);`
	mySqlExec(db, table)

	// таблица для хранения пользователей
	// session_id номер открытой смены
	// date дата открытой смены
	// fuelcons	NUMERIC(2,1)	расход топлива
	// fuelprice	NUMERIC(2,2)	стоимость топлива
	// workday	INTEGER	рабочих дней в месяце
	table = `CREATE TABLE IF NOT EXISTS users (
 users_id INTEGER PRIMARY KEY AUTOINCREMENT,
 name warchar(20),
 password TEXT,
 session_id INTEGER,
 date warchar(8),
 fuelcons NUMERIC(2,1),
 fuelprice NUMERIC(2,2),
 workday INTEGER);`
	mySqlExec(db, table)

	//таблица для хранения ремонтов
	// repair_id номер ремонта
	// user_id какому пользователю принадлежит
	// date дата ремонта
	// km пробег
	// detail ремонтируемый узел
	// cod код запчасти
	table = `CREATE TABLE IF NOT EXISTS repair (
 repair_id INTEGER PRIMARY KEY AUTOINCREMENT,
 user_id warchar(20),
 date warchar(8),
 km INTEGER,
 detail TEXT,
 cod TEXT);`
	mySqlExec(db, table)

	//проверяем, есть ли в таблице set данные в поле ver
	record, err := db.Query("SELECT ver FROM settings")
	if err != nil {
		log.Fatal(err)
	}
	defer record.Close()
	ver := 0
	for record.Next() {
		record.Scan(&ver)
		if err != nil {
			fmt.Println("Ошибка record.Scan")
			panic(err)
		}
	}
	if ver == 0 {
		//если нет данных в поле date, значит таблица только создана и в ней нет данных, добавляем их
		records := `INSERT INTO settings(ver, comdis, comper, comperter, comperonline) VALUES (2, 20, 170, 3, 3)`
		query, err := db.Prepare(records)
		if err != nil {
			log.Fatal(err)
		}
		_, err = query.Exec()
		if err != nil {
			log.Fatal(err)
		}
	}
} //2

func loadSettingsDB() {
	// читает основные переменные из БД
	now := time.Now()
	month := now.Format("01")
	var userid string
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	check(err)
	defer db.Close()

	record, err := db.Query("SELECT comdis, comper, comperter, comperonline FROM settings")
	check(err)
	defer record.Close()
	for record.Next() {
		record.Scan(&ComDis, &ComPer, &ComPerTer, &ComPerOnline)
		check(err)
	}

	// создает map с ключами users_id и значением у всех номер текущего месяца в string для использования в отчетах
	record, err = db.Query("SELECT users_id FROM users")
	check(err)
	defer record.Close()
	for record.Next() {
		record.Scan(&userid)
		check(err)
		Month[userid] = month
	}
} //2

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
} //2

func setDB() (string, string, string, string) {
	// возвращает настройки комиссий из БД таблицы settings
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	check(err)
	defer db.Close()

	record, err := db.Query("SELECT comdis, comper, comperter, comperonline FROM settings")
	check(err)
	defer record.Close()
	var comDis, comPer, comPerTer, comPerOnline string
	for record.Next() {
		record.Scan(&comDis, &comPer, &comPerTer, &comPerOnline)
		check(err)
	}
	return comDis, comPer, comPerTer, comPerOnline
} //2

func userSettingsDB(userid string) (float64, float64, int) {
	// возвращает расход, цену топлива и колич.рабочих дней в месяце пользователя
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	check(err)
	defer db.Close()

	record, err := db.Query("SELECT fuelcons, fuelprice, workday FROM users WHERE users_id = ?", userid)
	check(err)
	defer record.Close()
	var fuelCons, fuelPrise float64
	var workday int
	for record.Next() {
		record.Scan(&fuelCons, &fuelPrise, &workday)
		check(err)
	}
	return fuelCons, fuelPrise, workday
} //2

func userDB(userIdString string) (string, int) {
	// возвращает имя пользователя и id открытой смены по userId
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	check(err)
	defer db.Close()
	userId, _ := strconv.Atoi(userIdString)
	record, err := db.Query("SELECT name, session_id FROM users WHERE users_id = ?", userId)
	if err != nil {
		fmt.Println("func userDB: отсутствует user_id")
		return "", 0
	}
	defer record.Close()
	var name string
	var sessionId int
	for record.Next() {
		record.Scan(&name, &sessionId)
	}
	return name, sessionId
} //2

func smenaDB(userid string) []order_smena_text {
	// возвращает срез заказов за смену для userid
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	check(err)
	defer db.Close()

	record, err := db.Query("SELECT os_id, price, tea, type FROM orderssession WHERE user_id = ?;", userid)
	check(err)
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
} //2

func loadSessionDB(sessionid, userid string) []order_smena_text {
	// возвращает срез заказов для смены sessionId для userid из таблицы orders
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	check(err)
	defer db.Close()
	record, err := db.Query("SELECT orders_id, price, tea, type FROM orders WHERE user_id = ? AND session_id = ?;", userid, sessionid)
	check(err)
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

func kmhDB(userid string) []kmh_text {
	// возвращает срез смен из таблицы sessions со списком смен для пользователя userid
	month := Month[userid]
	monthSQL := fmt.Sprintf("__.%s.22", month)
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	check(err)
	defer db.Close()

	var fuel, fuelPrice, comDis, comPer, workDay float64
	record, err := db.Query("SELECT fuelcons, fuelprice, comdis, comper, workday FROM users, settings WHERE users_id = ?", userid)
	check(err)
	defer record.Close()
	for record.Next() {
		record.Scan(&fuel, &fuelPrice, &comDis, &comPer, &workDay)
		check(err)
	}
	comPer = comPer / workDay
	comPer = math.Round(comPer*10) / 10

	record, err = db.Query("SELECT session_id, date, km , h, price, tea, count FROM sessions INNER JOIN"+
		" (SELECT session_id AS orders_sessionid, sum(price) AS price, sum(tea) AS tea, count() AS count FROM orders"+
		" WHERE user_id = ? GROUP BY session_id) ON sessions.session_id = orders_sessionid WHERE date LIKE ? ORDER BY date", userid, monthSQL)
	check(err)

	smens := []kmh_text{}
	var num, km, count int
	var date string
	var h, price, tea float64
	smen := kmh_text{}
	for record.Next() {
		record.Scan(&num, &date, &km, &h, &price, &tea, &count)
		check(err)

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
} //2

func resetAutoIncrementOS() {
	//если таблица orderssession пустая, то сбрасывает счетчик AUTO_INCREMENT
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	check(err)
	defer db.Close()
	record, err := db.Query("SELECT count() FROM orderssession")
	check(err)
	defer record.Close()
	var count int
	for record.Next() {
		record.Scan(&count)
		check(err)
	}
	if count == 0 {
		records := `UPDATE SQLITE_SEQUENCE SET SEQ=0 WHERE NAME='orderssession'`
		query, err := db.Prepare(records)
		check(err)
		_, err = query.Exec()
		check(err)
	}
}

func indexNumDB() {
	// нумерует заказы в таблице smena по порядку
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	check(err)
	defer db.Close()

	// читаем в срез nums все записи поля num (номера заказа)
	record, err := db.Query("SELECT num FROM smena")
	check(err)
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
		check(err)
		_, err = query.Exec(i, nums[i-1])
		check(err)
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

func veryfUserDB(name string, pas string) (string, bool) {
	// возвращает true при совпадении пароля и userid из таблицы users
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	check(err)
	defer db.Close()
	record, err := db.Query("SELECT password, users_id FROM users WHERE name = ?", name)
	check(err)
	var password string
	var userid int
	for record.Next() {
		record.Scan(&password, &userid)
	}
	if userid == 0 {
		// нет такого имени
		return fmt.Sprint(userid), false
	}
	if GetMd5(pas) != password {
		// не соответствует пароль
		return "", false
	} else {
		return fmt.Sprint(userid), true
	}
} //2

func addUserDB(name string, pas string) string {
	// добавляет нового пользователя в БД в таблицу users
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	check(err)
	defer db.Close()
	hashPas := GetMd5(pas)
	records := `INSERT INTO users(name, password, session_id, date, fuelcons, fuelprice, workday)
 VALUES (?, ?, 0, "close", 12, 1.24, 24)`
	query, err := db.Prepare(records)
	check(err)
	_, err = query.Exec(name, hashPas)
	check(err)

	record, err := db.Query("SELECT users_id FROM users WHERE name = ?", name)
	check(err)
	var userid int
	for record.Next() {
		record.Scan(&userid)
	}
	return fmt.Sprint(userid)
} //2

func smenaSumTypDB(userid string, typ string) (string, string) {
	// возвращает сумму и количество заказов за смену по типу для userid в текстовом виде
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	check(err)
	defer db.Close()

	record, err := db.Query("SELECT sum(price), count() FROM orderssession WHERE type = ? AND user_id = ?", typ, userid)
	check(err)
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
} //2

func smenaSumDB(userid string) (string, string) {
	// возвращает сумму заказов за смену в текстовом виде "касса + чай = итог"
	// а вторым аргументом общее количество заказов за смену
	// для userid
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	check(err)
	defer db.Close()

	record, err := db.Query("SELECT sum(price), sum(tea), count() FROM orderssession WHERE user_id = ?", userid)
	check(err)
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
} //2

func addOrderDB(userid string, price string, tea string, typ string) {
	// добавляет заказ в БД в таблице orderssession
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	check(err)
	defer db.Close()

	records := `INSERT INTO orderssession(user_id, price, tea, type) VALUES (?, ?, ?, ?)`
	query, err := db.Prepare(records)
	check(err)
	_, err = query.Exec(userid, price, tea, typ)
	check(err)
} //2

func editOrderDB(userid string, osId string, price string, tea string, typ string) {
	// изменяет заказ номер num в БД в таблице orderssession
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	check(err)
	defer db.Close()

	records := `UPDATE orderssession SET price = ?, tea = ?, type = ? WHERE os_id = ? AND user_id = ?`
	query, err := db.Prepare(records)
	check(err)
	_, err = query.Exec(price, tea, typ, osId, userid)
	check(err)
} //2

func editOrdercloseDB(userid string, ordersid string, price string, tea string, typ string) {
	// изменяет заказ номер num в БД в таблице orders
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	check(err)
	defer db.Close()

	records := `UPDATE orders SET price = ?, tea = ?, type = ? WHERE orders_id = ? AND user_id = ?`
	query, err := db.Prepare(records)
	check(err)
	_, err = query.Exec(price, tea, typ, ordersid, userid)
	check(err)
}

func addOrdercloseDB(userid, sessionid, price, tea, typ string) {
	// добавляет заказ в БД в таблицу orders
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	check(err)
	defer db.Close()

	records := `INSERT INTO orders(user_id, session_id, price, tea, type) VALUES (?, ?, ?, ?, ?)`
	query, err := db.Prepare(records)
	check(err)
	_, err = query.Exec(userid, sessionid, price, tea, typ)
	check(err)
}

func delOrderDB(userid string, osId int) {
	// удаление заказа n для userid из таблицы orderssession
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	check(err)
	defer db.Close()

	records := `DELETE FROM orderssession WHERE os_id = ? AND user_id = ?`
	query, err := db.Prepare(records)
	check(err)
	_, err = query.Exec(osId, userid)
	check(err)
} //2

func delOrderCloseDB(orderid, userid string) {
	// удаление заказа orderid для userid из таблицы orders
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	check(err)
	defer db.Close()

	records := `DELETE FROM orders WHERE orders_id = ? AND user_id = ?`
	query, err := db.Prepare(records)
	check(err)
	_, err = query.Exec(orderid, userid)
	check(err)
}

func dateOpenSessionDB(userid string) string {
	// возвращает дату открытой смены из таблицы users по userid. "close" если закрыта
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	check(err)
	defer db.Close()

	record, err := db.Query("SELECT date FROM users WHERE users_id = ?", userid)
	check(err)
	defer record.Close()
	date := ""
	for record.Next() {
		record.Scan(&date)
		check(err)
	}
	return date
} //2

func dateToUsersDB(userid string, date string) {
	//изменяет в таблице users поле date на date ("close" значит, что сейчас смена закрыта)
	// в поле session_id пишет номер смены или 0
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	check(err)
	defer db.Close()

	var lastsession int
	if date != "close" {
		record, err := db.Query("SELECT lastsession FROM settings")
		check(err)
		for record.Next() {
			record.Scan(&lastsession)
		}
		lastsession++

		records := `UPDATE settings SET lastsession = ?`
		query, err := db.Prepare(records)
		check(err)
		_, err = query.Exec(lastsession)
		check(err)
	}

	records := `UPDATE users SET date = ?, session_id = ?  WHERE users_id = ?`
	query, err := db.Prepare(records)
	check(err)
	_, err = query.Exec(date, lastsession, userid)
	check(err)
} //2

func addSessionsDB(userid string, date string, km int, h float64) {
	// добавляет итоги смены в таблицу sessions
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	check(err)
	defer db.Close()

	var sessionid int
	record, err := db.Query("SELECT session_id FROM users WHERE users_id = ?", userid)
	check(err)
	for record.Next() {
		record.Scan(&sessionid)
	}

	records := `INSERT INTO sessions(session_id, date, user_id, km, h) VALUES (?, ?, ?, ?, ?)`
	query, err := db.Prepare(records)
	check(err)
	_, err = query.Exec(sessionid, date, userid, km, h)
	check(err)
	// indexNumSmenDB()
} //2

func smenaTOordersDB(userid string, date string) {
	// переносит данные из таблицы sessions в orders для userid
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	check(err)
	defer db.Close()
	records := `INSERT INTO orders (user_id, session_id, price, tea, type) SELECT user_id, session_id, price, tea, type FROM users INNER JOIN orderssession ON users.users_id = orderssession.user_id WHERE users_id = ?`
	query, err := db.Prepare(records)
	check(err)
	_, err = query.Exec(userid)
	check(err)

	// очищает таблицу smena
	records = `DELETE FROM orderssession WHERE user_id = ?`
	query, err = db.Prepare(records)
	check(err)
	_, err = query.Exec(userid)
	check(err)
} //2

func distanceHDB(userid string, month string) (int, float64, int) {
	// возвращает пробег, время и количество смен за месяц mounth для userid
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	check(err)
	defer db.Close()
	monthSQL := fmt.Sprintf("__.%s.22", month)
	record, err := db.Query("SELECT sum(km), sum(h), count() FROM sessions WHERE date LIKE ? AND user_id = ?", monthSQL, userid)
	check(err)
	defer record.Close()
	var h float64 // для суммы часов
	var km int    // для суммы километров
	var count int // для количества смен
	for record.Next() {
		record.Scan(&km, &h, &count)
		check(err)
	}
	return km, h, count
} //2

func reportDB(month string, userid string, typ string) (float64, int, []string) {
	// возвращает сумму заказов за месяц month по типу typ в текстовом виде в sum
	// количество в cont и список заказов в orders для userid
	var record *sql.Rows
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	check(err)
	defer db.Close()
	monthSQL := fmt.Sprintf("__.%s.22", month)
	if typ == "all" {
		record, err = db.Query("SELECT sum(price), count() FROM sessions INNER JOIN orders ON sessions.session_id = orders.session_id WHERE sessions.date LIKE ? AND sessions.user_id = ?", monthSQL, userid)
		check(err)
	} else {
		record, err = db.Query("SELECT sum(price), count() FROM sessions INNER JOIN orders ON sessions.session_id = orders.session_id WHERE sessions.date LIKE ? AND sessions.user_id = ? AND orders.type = ?", monthSQL, userid, typ)
		check(err)
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
	var orders []string
	if typ != "all" {
		record, err = db.Query("SELECT date, price, tea FROM sessions INNER JOIN orders ON sessions.session_id = orders.session_id WHERE sessions.date LIKE ? AND orders.type = ? AND sessions.user_id = ? ORDER BY sessions.date", monthSQL, typ, userid)
		check(err)
		var date, s string
		var price, tea float64
		var n int

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
	}
	return sum, count, orders
} //2

func delSmenDB(userid string, sessionId int) {
	// удаление смены sessionId для userid из БД sessions
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	check(err)
	defer db.Close()

	records := `DELETE FROM sessions WHERE session_id = ? AND user_id = ?`
	query, err := db.Prepare(records)
	check(err)
	_, err = query.Exec(sessionId, userid)
	check(err)

	// удаляем заказы с sessionId и userid из таблицы orders
	records = `DELETE FROM orders WHERE session_id = ? AND user_id = ?`
	query, err = db.Prepare(records)
	check(err)
	_, err = query.Exec(sessionId, userid)
	check(err)
	//indexNumSmenDB()
} //2

func editSmenDB(userid string, sessionId string, km string, h string) {
	// изменяет смену номер sessionId в БД в таблице sessions
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	check(err)
	defer db.Close()

	records := `UPDATE sessions SET km = ?, h = ? WHERE session_id = ? AND user_id = ?`
	query, err := db.Prepare(records)
	check(err)
	_, err = query.Exec(km, h, sessionId, userid)
	check(err)
} //2

func addRepairDB(userid, detail, cod, date, km string) {
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	check(err)
	defer db.Close()

	records := `INSERT INTO repair (user_id, date, km, detail, cod) VALUES (?, ?, ?, ?, ?)`
	query, err := db.Prepare(records)
	check(err)
	_, err = query.Exec(userid, date, km, detail, cod)
	check(err)
}

func delRepairDB(userid, repair_id string) {
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	check(err)
	defer db.Close()

	records := `DELETE FROM repair WHERE repair_id = ? AND user_id = ?`
	query, err := db.Prepare(records)
	check(err)
	_, err = query.Exec(repair_id, userid)
	check(err)
}

func loadrepairDB(userid, id string) repair_form_type {
	var out repair_form_type
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	check(err)
	defer db.Close()

	record, err := db.Query("SELECT date, km, detail, cod FROM repair WHERE repair_id = ? AND user_id = ?", id, userid)
	check(err)
	defer record.Close()
	for record.Next() {
		record.Scan(&out.Date, &out.Km, &out.Detail, &out.Cod)
		check(err)
	}
	out.Id = id
	return out
}

func editRepairDB(userid, repair_id, detail, cod, date, km string) {
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	check(err)
	defer db.Close()

	records := `UPDATE repair SET date = ?, km = ?, detail = ?, cod = ? WHERE repair_id = ? AND user_id = ?`
	query, err := db.Prepare(records)
	check(err)
	_, err = query.Exec(date, km, detail, cod, repair_id, userid)
	check(err)
}

func repairDB(userid string) []details_type {
	var details []details_type
	var i_details int = -1 // номер элемента в срезе для lastDetail
	var tempDetail details_type
	var curentDetail, lastDetail string
	var repair repairs_type
	db, err := sql.Open("sqlite3", "../dir_db/taxi.db")
	check(err)
	defer db.Close()

	record, err := db.Query("SELECT repair_id, date, km, detail, cod FROM repair WHERE user_id = ? ORDER BY detail, km DESC", userid)
	check(err)
	defer record.Close()
	for record.Next() {
		record.Scan(&repair.Repair_id, &repair.Date, &repair.Km, &curentDetail, &repair.Cod)
		check(err)
		if lastDetail != curentDetail {
			// началась новая группа деталей(узлов) поэтому создаем новый элемент среза и добавляем туда первую запись
			tempDetail.Detail = curentDetail
			tempDetail.Date = repair.Date
			tempDetail.Passed = repair.Km // пока здесь километраж при ремонте, потом отнимем текущий и запишем сюда разницу
			tempDetail.Repairs = append(tempDetail.Repairs, repair)
			details = append(details, tempDetail)
			tempDetail = details_type{}
			lastDetail = curentDetail
			i_details++
		} else {
			// продолжается прочитанная прошлый раз группа, просто добавляем ремонт
			details[i_details].Repairs = append(details[i_details].Repairs, repair)
		}
	}
	return details
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
} //2

func smena(w http.ResponseWriter, r *http.Request) {
	userCookie, err := r.Cookie("userid")
	if err != nil {
		// нет кука userid, надо залогинится
		// переходим на страницу логина
		t, err := template.ParseFiles("templates/login.html",
			"templates/header.html", "templates/footer.html")
		check(err)
		err = t.ExecuteTemplate(w, "login", "0")
		check(err)
	} else {
		// в куках есть userid
		userid := userCookie.Value
		date := dateOpenSessionDB(userid)
		if date == "close" {
			// если смена закрыта
			t, err := template.ParseFiles("./templates/closesmene.html",
				"./templates/header.html", "./templates/footer.html")
			check(err)
			var outt close_smena
			outt.UserName, _ = userDB(userid)
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
			var sessionId int
			out.UserName, sessionId = userDB(userid)
			out.SessionId = fmt.Sprint(sessionId)
			out.Date = date
			out.Order = smenaDB(userid)
			out.NalSum, out.NalCount = smenaSumTypDB(userid, "n")
			out.TermSum, out.TermCount = smenaSumTypDB(userid, "t")
			out.OnlineSum, out.OnlineCount = smenaSumTypDB(userid, "o")
			out.KampSum, out.KampCount = smenaSumTypDB(userid, "k")
			out.TotalSum, out.TotalCount = smenaSumDB(userid)
			out.Coment = "ok"
			out.IndexComment = IndexComment[userid]

			t.ExecuteTemplate(w, "index", out)
		}
	}
} //2

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
		userCookie, err := r.Cookie("userid")
		check(err)
		userid := userCookie.Value
		dateToUsersDB(userid, date)

		defer smena(w, r)
	} else {
		defer errorDate(w, r)
	}
} //2

func addorder(w http.ResponseWriter, r *http.Request) {
	in := r.FormValue("in")
	if in != "" {
		userCookie, err := r.Cookie("userid")
		check(err)
		userid := userCookie.Value
		fmt.Print("userid=" + userid + " вводит ")
		inSl := okOrder(in)
		addOrderDB(userid, inSl[1], inSl[2], inSl[0])
	}
	defer smena(w, r)
} //2

func delOrder(w http.ResponseWriter, r *http.Request) {
	userCookie, err := r.Cookie("userid")
	check(err)
	userid := userCookie.Value
	in := r.FormValue("in")
	i, err := strconv.Atoi(in)
	if err == nil {
		delOrderDB(userid, i)
	}
	defer corect(w, r)
} //2

func editOrder(w http.ResponseWriter, r *http.Request) {
	userCookie, err := r.Cookie("userid")
	check(err)
	userid := userCookie.Value
	// номер редактируемого заказа
	num := r.FormValue("num")
	// новые данные заказа
	edit := r.FormValue("edit")
	inSl := okOrder(edit)
	editOrderDB(userid, num, inSl[1], inSl[2], inSl[0])

	defer corect(w, r)
} //2

func corect(w http.ResponseWriter, r *http.Request) {
	userCookie, err := r.Cookie("userid")
	if err != nil {
		// нет кука userid, надо залогинится
		// переходим на страницу логина
		t, err := template.ParseFiles("templates/login.html",
			"templates/header.html", "templates/footer.html")
		check(err)
		err = t.ExecuteTemplate(w, "login", "0")
		check(err)
	} else {
		// в куках есть userid
		userid := userCookie.Value
		t, err := template.ParseFiles("./templates/corect.html",
			"./templates/header.html", "./templates/footer.html")
		check(err)

		var out smen
		var sessionId int
		out.UserName, sessionId = userDB(userid)
		out.SessionId = fmt.Sprint(sessionId)
		out.Order = smenaDB(userid)
		out.NalSum, out.NalCount = smenaSumTypDB(userid, "n")
		out.TermSum, out.TermCount = smenaSumTypDB(userid, "t")
		out.OnlineSum, out.OnlineCount = smenaSumTypDB(userid, "o")
		out.KampSum, out.KampCount = smenaSumTypDB(userid, "k")
		out.TotalSum, out.TotalCount = smenaSumDB(userid)
		out.Coment = "ok"

		t.ExecuteTemplate(w, "corect", out)
	}
} //2

func sclose(w http.ResponseWriter, r *http.Request) {
	// закрывает смену
	// удаляе заказы из таблицы orderssession для userid
	// удаляет дату смены из таблицы users меняя на "close"
	t, err := template.ParseFiles("./templates/close.html",
		"./templates/header.html", "./templates/footer.html")
	if err != nil {
		fmt.Println("ошибка template.ParseFiles")
		fmt.Println(err.Error())
	}
	userCookie, err := r.Cookie("userid")
	check(err)
	userid := userCookie.Value
	var out smen
	var sessionId int
	out.UserName, sessionId = userDB(userid)
	out.SessionId = fmt.Sprint(sessionId)
	out.Date = dateOpenSessionDB(userid)
	out.NalSum, out.NalCount = smenaSumTypDB(userid, "n")
	out.TermSum, out.TermCount = smenaSumTypDB(userid, "t")
	out.OnlineSum, out.OnlineCount = smenaSumTypDB(userid, "o")
	out.KampSum, out.KampCount = smenaSumTypDB(userid, "k")
	out.TotalSum, out.TotalCount = smenaSumDB(userid)
	out.Coment = "ok"
	count1, _ := strconv.Atoi(out.NalCount)
	count2, _ := strconv.Atoi(out.TermCount)
	count := count1 + count2
	out.NalTermCount = fmt.Sprintf("%d", count)

	t.ExecuteTemplate(w, "close", out)
} //2

func scloseForm(w http.ResponseWriter, r *http.Request) {
	userCookie, err := r.Cookie("userid")
	check(err)
	userid := userCookie.Value
	kms := r.FormValue("km")
	km, _ := strconv.Atoi(kms)
	hs := r.FormValue("h")
	h, _ := strconv.ParseFloat(hs, 64)
	date := dateOpenSessionDB(userid)
	addSessionsDB(userid, date, km, h) // добавляет итоги смены в таблицу sessions
	fmt.Println("userid="+userid, " Закрытие смены: ", km, "км, ", h, "ч")
	smenaTOordersDB(userid, date)
	resetAutoIncrementOS()
	dateToUsersDB(userid, "close")
	defer smena(w, r)
} //2

func report(w http.ResponseWriter, r *http.Request) {
	userCookie, err := r.Cookie("userid")
	if err != nil {
		// нет кука userid, надо залогинится
		// переходим на страницу логина
		t, err := template.ParseFiles("templates/login.html",
			"templates/header.html", "templates/footer.html")
		check(err)
		err = t.ExecuteTemplate(w, "login", "0")
		check(err)
	} else {
		// в куках есть userid
		userid := userCookie.Value
		var nalSum, termSum, onlineSum, kampSum, totalSum float64
		var nalCount, termCount, onlineCount, kampCount, totalCount int
		t, err := template.ParseFiles("./templates/report.html",
			"./templates/header.html", "./templates/footer.html")
		if err != nil {
			fmt.Println("ошибка template.ParseFiles")
			fmt.Println(err.Error())
		}
		out := rep{}
		out.Month = Month[userid]
		out.UserName, _ = userDB(userid)
		nalSum, nalCount, out.Nal = reportDB(out.Month, userid, "n")
		out.NalSum = fmt.Sprintf("%.2f", nalSum)
		out.NalCount = fmt.Sprintf("%d", nalCount)
		termSum, termCount, out.Term = reportDB(out.Month, userid, "t")
		out.TermSum = fmt.Sprintf("%.2f", termSum)
		out.TermCount = fmt.Sprintf("%d", termCount)
		onlineSum, onlineCount, out.Online = reportDB(out.Month, userid, "o")
		out.OnlineSum = fmt.Sprintf("%.2f", onlineSum)
		out.OnlineCount = fmt.Sprintf("%d", onlineCount)
		kampSum, kampCount, out.Kamp = reportDB(out.Month, userid, "k")
		out.KampSum = fmt.Sprintf("%.2f", kampSum)
		out.KampCount = fmt.Sprintf("%d", kampCount)
		totalSum, totalCount, _ = reportDB(out.Month, userid, "all")
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
		out.PayKamp = fmt.Sprintf("%.2f", kampSum)
		paySum := payTerm + payOnline + kampSum
		out.PaySum = fmt.Sprintf("%.2f", paySum)
		out.Balance = fmt.Sprintf("%.2f", comSum-paySum)
		distance, hours, count := distanceHDB(userid, out.Month)
		out.Distance = fmt.Sprintf("%.d", distance)
		out.Hours = fmt.Sprintf("%.2f", hours)
		out.Count = fmt.Sprintf("%.d", count)
		fuelCons, fuelPrise, _ := userSettingsDB(userid)
		payDist := fuelCons / 100 * float64(distance) * fuelPrise
		out.PayDist = fmt.Sprintf("%.0f", payDist)
		salary := totalSum - payDist - comSum
		out.Salary = fmt.Sprintf("%.0f", salary)
		out.SalaryH = fmt.Sprintf("%.1f", salary/hours)
		t.ExecuteTemplate(w, "report", out)
	}
} //2

func kmh(w http.ResponseWriter, r *http.Request) {
	userCookie, err := r.Cookie("userid")
	check(err)
	userid := userCookie.Value
	m, err := strconv.Atoi(Month[userid])
	check(err)
	z := r.FormValue("znak")
	if z == "-" {
		if m == 1 {
			m = 12
		} else {
			m--
		}
		if m < 10 {
			Month[userid] = "0" + fmt.Sprint(m)
		} else {
			Month[userid] = fmt.Sprint(m)
		}
	}
	if z == "+" {
		if m == 12 {
			m = 1
		} else {
			m++
		}
		if m < 10 {
			Month[userid] = "0" + fmt.Sprint(m)
		} else {
			Month[userid] = fmt.Sprint(m)
		}
	}

	t, err := template.ParseFiles("./templates/kmh.html",
		"./templates/header.html", "./templates/footer.html")
	check(err)
	out := kmhDB(userid)
	t.ExecuteTemplate(w, "kmh", out)
} //2

func delSmen(w http.ResponseWriter, r *http.Request) {
	userCookie, err := r.Cookie("userid")
	check(err)
	userid := userCookie.Value
	in := r.FormValue("in")
	i, err := strconv.Atoi(in)
	if err == nil {
		delSmenDB(userid, i)
	}
	defer kmh(w, r)
} //2

func editSmen(w http.ResponseWriter, r *http.Request) {
	userCookie, err := r.Cookie("userid")
	check(err)
	userid := userCookie.Value
	var er int
	// номер редактируемой смены
	num := r.FormValue("num")
	// новый километраж за смену
	km := r.FormValue("km")
	i, err := strconv.Atoi(km)
	if err != nil {
		er++
		i++
	}
	// новое время смены
	h := r.FormValue("h")
	fl, err := strconv.ParseFloat(h, 64)
	if err != nil {
		er++
		fl++
	}
	if er == 0 {
		editSmenDB(userid, num, km, h)
	}
	defer kmh(w, r)
} //2

func saveDb(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("./templates/settings.html",
		"./templates/header.html", "./templates/footer.html")
	if err != nil {
		fmt.Println("ошибка template.ParseFiles")
		fmt.Println(err.Error())
	}
	var out Sett

	userCookie, err := r.Cookie("userid")
	check(err)
	userid := userCookie.Value
	out.UserName, _ = userDB(userid)

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

	fuelCons, fuelPrice, workDay := userSettingsDB(userid)
	out.FuelCons = fmt.Sprint(fuelCons)
	out.FuelPrice = fmt.Sprint(fuelPrice)
	out.WorkDay = fmt.Sprint(workDay)
	out.ComDis, out.ComPer, out.ComPerTer, out.ComPerOnline = setDB()
	t.ExecuteTemplate(w, "settings", out)
}

func errorDate(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("./templates/closesmene.html",
		"./templates/header.html", "./templates/footer.html")
	check(err)
	var out close_smena
	userCookie, err := r.Cookie("userid")
	check(err)
	userid := userCookie.Value
	out.UserName, _ = userDB(userid)
	now := time.Now()
	out.Date = now.Format("02.01.06")
	out.Ok = "err"
	t.ExecuteTemplate(w, "closesmene", out)
} //2

func userExit(w http.ResponseWriter, r *http.Request) {
	c := &http.Cookie{
		Name:     "userid",
		Value:    "",
		Expires:  time.Unix(0, 0),
		HttpOnly: true,
	}
	http.SetCookie(w, c)
	w.Header().Set("Location", "/")
	w.Header().Set("Cache-Control", "private, no-store, max-age=0, must-revalidate")
	w.WriteHeader(303)
} //2

func veryfUser(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	inPas := r.FormValue("password")
	fmt.Println("введено имя:", name, ", пароль: ", inPas)

	userid, ok := veryfUserDB(name, inPas)
	if !ok || name == "" || inPas == "" {
		fmt.Println("не совпал пароль: ")
		t, err := template.ParseFiles("templates/login.html",
			"templates/header.html", "templates/footer.html")
		check(err)
		t.ExecuteTemplate(w, "login", "1")
	} else {
		// АВТОРИЗУЕМ ПОЛЬЗОВАТЕЛЯ
		c := &http.Cookie{
			Name:  "userid",
			Value: userid,
		}
		http.SetCookie(w, c)
		fmt.Println(name, "успешно залогинен c user_id =", userid)
		w.Header().Set("Location", "/")
		w.Header().Set("Cache-Control", "private, no-store, max-age=0, must-revalidate")
		w.WriteHeader(303)
	}
} //2

func addNewUser(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	inPas := r.FormValue("password")
	fmt.Println("Создание нового пользователя!")
	fmt.Println("login.html введено имя:", name, ", пароль: ", inPas)
	userid, _ := veryfUserDB(name, inPas)
	if userid != "0" {
		fmt.Println("пользователь: ", name, " уже существует ")
		t, err := template.ParseFiles("templates/login.html",
			"templates/header.html", "templates/footer.html")
		check(err)
		err = t.ExecuteTemplate(w, "login", "2")
		check(err)
	} else if name == "" || inPas == "" {
		fmt.Println("пустое имя или пароль")
		t, err := template.ParseFiles("templates/login.html",
			"templates/header.html", "templates/footer.html")
		check(err)
		t.ExecuteTemplate(w, "login", "3")

	} else {
		fmt.Println("создаем нового пользователя в БД")
		userid := addUserDB(name, inPas)
		c := &http.Cookie{
			Name:  "userid",
			Value: userid,
		}
		http.SetCookie(w, c)
		fmt.Println(name, "успешно создан c user_id =", userid)
		w.Header().Set("Location", "/")
		w.Header().Set("Cache-Control", "private, no-store, max-age=0, must-revalidate")
		w.WriteHeader(303)
	}
} //2

func settings(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("./templates/settings.html",
		"./templates/header.html", "./templates/footer.html")
	check(err)
	userCookie, err := r.Cookie("userid")
	check(err)
	userid := userCookie.Value
	var out Sett
	out.UserName, _ = userDB(userid)
	fuelCons, fuelPrice, workDay := userSettingsDB(userid)
	out.FuelCons = fmt.Sprint(fuelCons)
	out.FuelPrice = fmt.Sprint(fuelPrice)
	out.WorkDay = fmt.Sprint(workDay)
	out.ComDis, out.ComPer, out.ComPerTer, out.ComPerOnline = setDB()
	out.Coment = "ok"
	t.ExecuteTemplate(w, "settings", out)
} //2

func setSettings(w http.ResponseWriter, r *http.Request) {
	userCookie, err := r.Cookie("userid")
	check(err)
	userid := userCookie.Value
	var in Sett
	in.UserName = userid // здесь в in.UserName - userid
	in.FuelCons = r.FormValue("fuelcons")
	in.FuelPrice = r.FormValue("fuelprice")
	in.WorkDay = r.FormValue("workday")
	in.ComDis = r.FormValue("comdis")
	in.ComPer = r.FormValue("comper")
	in.ComPerTer = r.FormValue("comperter")
	in.ComPerOnline = r.FormValue("comperonline")
	fmt.Println(in)
	setSettingsDB(in)
	defer settings(w, r)
} //2

func MonthMinus(w http.ResponseWriter, r *http.Request) {
	userCookie, err := r.Cookie("userid")
	check(err)
	userid := userCookie.Value
	m, err := strconv.Atoi(Month[userid])
	check(err)
	if m == 1 {
		m = 12
	} else {
		m--
	}
	if m < 10 {
		Month[userid] = "0" + fmt.Sprint(m)
	} else {
		Month[userid] = fmt.Sprint(m)
	}
	defer report(w, r)
}

func MonthPlus(w http.ResponseWriter, r *http.Request) {
	userCookie, err := r.Cookie("userid")
	check(err)
	userid := userCookie.Value
	m, err := strconv.Atoi(Month[userid])
	check(err)
	if m == 12 {
		m = 1
	} else {
		m++
	}
	if m < 10 {
		Month[userid] = "0" + fmt.Sprint(m)
	} else {
		Month[userid] = fmt.Sprint(m)
	}
	defer report(w, r)
}

func differenceKm(details []details_type, km string) []details_type {
	var tempKm int
	curKm, err := strconv.Atoi(km)
	check(err)

	for i := range details {
		tempKm, err = strconv.Atoi(details[i].Passed)
		check(err)
		details[i].Passed = fmt.Sprint((curKm - tempKm) / 1000)
	}
	return details
}

func editcomment(w http.ResponseWriter, r *http.Request) {
	userCookie, err := r.Cookie("userid")
	check(err)
	userid := userCookie.Value
	// todo добавить проверку, что есть куки userid
	com := r.FormValue("incomment")
	IndexComment[userid] = com
	defer smena(w, r)
}

func repair(w http.ResponseWriter, r *http.Request) {
	userCookie, err := r.Cookie("userid")
	check(err)
	userid := userCookie.Value
	var out repair_type
	out.UserName, _ = userDB(userid)

	km := ""
	km = r.FormValue("ckm")
	if km != "" {
		// на странице введен пробег
		if km == " " {
			km = ""
		}
		Km[userid] = km
		out.CurKm = km
	} else {
		// пробег не вводили. читаем текущий
		out.CurKm = Km[userid]
	}

	t, err := template.ParseFiles("templates/repair.html",
		"templates/header.html", "templates/footer.html")
	check(err)

	tempDetails := repairDB(userid)
	if out.CurKm == "" {
		out.Details = tempDetails
	} else {
		out.Details = differenceKm(tempDetails, out.CurKm)
	}
	err = t.ExecuteTemplate(w, "repair", out)
	check(err)
}

func addRepair(w http.ResponseWriter, r *http.Request) {
	userCookie, err := r.Cookie("userid")
	check(err)
	userid := userCookie.Value
	detail := r.FormValue("detail")
	cod := r.FormValue("cod")
	date := r.FormValue("date")
	km := r.FormValue("km")
	if detail != "" || cod != "" || date != "" || km != "" {
		addRepairDB(userid, detail, cod, date, km)
	}
	defer repair(w, r)
}

func loadRepair(w http.ResponseWriter, r *http.Request) {
	var out repair_type
	userCookie, err := r.Cookie("userid")
	check(err)
	userid := userCookie.Value
	repair_id := r.FormValue("repair_id")
	if repair_id != "" {
		out.Form = loadrepairDB(userid, repair_id)
	}
	out.UserName, _ = userDB(userid)
	out.CurKm = Km[userid]
	out.Details = repairDB(userid)
	t, err := template.ParseFiles("templates/repair.html",
		"templates/header.html", "templates/footer.html")
	check(err)
	err = t.ExecuteTemplate(w, "repair", out)
	check(err)
}

func editRepair(w http.ResponseWriter, r *http.Request) {
	userCookie, err := r.Cookie("userid")
	check(err)
	userid := userCookie.Value
	repair_id := r.FormValue("repair_id")
	detail := r.FormValue("detail")
	cod := r.FormValue("cod")
	date := r.FormValue("date")
	km := r.FormValue("km")
	if detail != "" || cod != "" || date != "" || km != "" {
		editRepairDB(userid, repair_id, detail, cod, date, km)
	}
	defer repair(w, r)
}

func delRepair(w http.ResponseWriter, r *http.Request) {
	userCookie, err := r.Cookie("userid")
	check(err)
	userid := userCookie.Value
	repair_id := r.FormValue("repair_id")
	if repair_id != "" {
		delRepairDB(userid, repair_id)
	}
	defer repair(w, r)
}

func loadSession(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/editord.html",
		"templates/header.html", "templates/footer.html")
	check(err)
	userCookie, err := r.Cookie("userid")
	check(err)
	userid := userCookie.Value
	sessionId := r.FormValue("in")
	var out smen
	out.UserName, _ = userDB(userid)
	out.SessionId = sessionId
	out.Order = loadSessionDB(sessionId, userid)

	err = t.ExecuteTemplate(w, "editord", out)
	check(err)
}

func delorderclose(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/editord.html",
		"templates/header.html", "templates/footer.html")
	check(err)
	userCookie, err := r.Cookie("userid")
	check(err)
	userid := userCookie.Value
	orderId := r.FormValue("in")
	sessionId := r.FormValue("sessionid")
	delOrderCloseDB(orderId, userid)
	var out smen
	out.UserName, _ = userDB(userid)
	out.SessionId = sessionId
	out.Order = loadSessionDB(sessionId, userid)
	err = t.ExecuteTemplate(w, "editord", out)
	check(err)
}

func editorderclose(w http.ResponseWriter, r *http.Request) {
	t, err := template.ParseFiles("templates/editord.html",
		"templates/header.html", "templates/footer.html")
	check(err)
	userCookie, err := r.Cookie("userid")
	check(err)
	userid := userCookie.Value
	sessionId := r.FormValue("sessionid")
	num := r.FormValue("num")
	edit := r.FormValue("edit")
	inSl := okOrder(edit)
	if num != "" {
		editOrdercloseDB(userid, num, inSl[1], inSl[2], inSl[0])
	} else {
		addOrdercloseDB(userid, sessionId, inSl[1], inSl[2], inSl[0])
	}
	var out smen
	out.UserName, _ = userDB(userid)
	out.SessionId = sessionId
	out.Order = loadSessionDB(sessionId, userid)
	err = t.ExecuteTemplate(w, "editord", out)
	check(err)
}

func main() {
	ver := versionDB(0)
	fmt.Println("версия БД: ", ver)
	if ver == 0 {
		fmt.Println("Обнаружена старая БД. Трансформирую в версию: 2")
		transformDBto2()
	} else if ver == -1 {
		fmt.Println("База данных taxi.db не найдена. Создана пустая")
		createTablesDB()
	} else if ver == 2 {
		fmt.Println("Обновляется до Ver 3 (добавлена таблица repair)")
		versionDB(3)
		createTablesDB()
	}
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
	http.HandleFunc("/user_exit", userExit)
	http.HandleFunc("/verif_user", veryfUser)
	http.HandleFunc("/add_new_user", addNewUser)
	http.HandleFunc("/settings", settings)
	http.HandleFunc("/set_settings", setSettings)
	http.HandleFunc("/report_minus", MonthMinus)
	http.HandleFunc("/report_plus", MonthPlus)
	http.HandleFunc("/repair", repair)
	http.HandleFunc("/add_repair", addRepair)
	http.HandleFunc("/load_repair", loadRepair)
	http.HandleFunc("/edit_repair", editRepair)
	http.HandleFunc("/del_repair", delRepair)
	http.HandleFunc("/load_session", loadSession)
	http.HandleFunc("/delorder", delorderclose)
	http.HandleFunc("/editorder", editorderclose)
	http.HandleFunc("/edit_comment", editcomment)

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static/"))))
	http.Handle("/dir_db/", http.StripPrefix("/dir_db/", http.FileServer(http.Dir("../dir_db/"))))
	log.Println(http.ListenAndServe(":5005", nil))
}
