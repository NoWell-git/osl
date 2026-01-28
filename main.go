package main

import (
	"bufio"
	"database/sql"
	"fmt"
	"log"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// Структура для хранения информации о таблице
type TableInfo struct {
	Name    string
	Columns []string
}

// Структура для конфигурации БД
type DBConfig struct {
	Host     string
	Port     string
	Name     string
	User     string
	Password string
	SSLMode  string
}

// Глобальные переменные
var (
	db             *sql.DB
	tables         []TableInfo
	relatedTables  []string
	logFile        *os.File
	whiteListRegex = regexp.MustCompile(`^[a-zA-Zа-яА-ЯёЁ0-9\s\-\.]+$`)
)

func main() {
	// Получение пути к файлу логов из переменной окружения
	logPath := os.Getenv("LOG_FILE")
	if logPath == "" {
		logPath = "/logs/app.log"
	}

	// Создание директории для логов если не существует
	os.MkdirAll("/logs", 0755)

	// Открытие файла логов
	var err error
	logFile, err = os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Ошибка открытия файла логов: %v", err)
		os.Exit(1)
	}
	defer logFile.Close()

	// Настройка логгера для записи в файл
	log.SetOutput(logFile)

	fmt.Println("=== Подключение к базе данных ===")

	// Запрос учетных данных у пользователя
	reader := bufio.NewReader(os.Stdin)

	fmt.Print("Введите логин: ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	fmt.Print("Введите пароль: ")
	password, _ := reader.ReadString('\n')
	password = strings.TrimSpace(password)

	// Чтение конфигурации из переменных окружения
	config := DBConfig{
		Host:     os.Getenv("DB_HOST"),
		Port:     os.Getenv("DB_PORT"),
		Name:     os.Getenv("DB_NAME"),
		User:     username,
		Password: password,
		SSLMode:  os.Getenv("DB_SSLMODE"),
	}

	// Подключение к базе данных
	connectionString := fmt.Sprintf("host=%s port=%s dbname=%s user=%s password=%s sslmode=%s",
		config.Host, config.Port, config.Name, config.User, config.Password, config.SSLMode)

	var connectErr error
	db, connectErr = sql.Open("postgres", connectionString)
	if connectErr != nil {
		logToFileAndScreen(fmt.Sprintf("Ошибка подключения к БД: %v", connectErr))
		fmt.Println("Ошибка: Не удалось подключиться к базе данных. Проверьте учетные данные.")
		os.Exit(1)
	}

	// Ждем запуска PostgreSQL
	logToFileAndScreen("Ожидание запуска PostgreSQL...")
	time.Sleep(5 * time.Second)

	// Проверка подключения с повторными попытками
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		if err := db.Ping(); err != nil {
			logToFileAndScreen(fmt.Sprintf("Попытка %d: Ошибка проверки подключения: %v", i+1, err))
			if i < maxRetries-1 {
				time.Sleep(2 * time.Second)
				continue
			}
			logToFileAndScreen("Ошибка: Не удалось подключиться к базе данных")
			fmt.Println("Ошибка: Не удалось подключиться к базе данных. Проверьте учетные данные и доступность БД.")
			os.Exit(1)
		}
		break
	}

	logToFileAndScreen("Успешное подключение к базе данных")
	fmt.Println("✓ Подключение к базе данных успешно установлено")

	// Загрузка информации о таблицах
	loadTableInfo()

	// Определение связанных таблиц
	relatedTables = []string{
		"components и stock",
		"categories и components",
		"manufacturers и components",
	}

	// Запуск главного меню
	mainMenu(reader)
}

// Функция для загрузки информации о таблицах
func loadTableInfo() {
	tables = []TableInfo{
		{Name: "categories", Columns: []string{"id", "name", "description"}},
		{Name: "manufacturers", Columns: []string{"id", "name", "country", "founded_year"}},
		{Name: "components", Columns: []string{"id", "name", "category_id", "manufacturer_id", "model", "price"}},
		{Name: "stock", Columns: []string{"id", "component_id", "quantity", "warehouse_location"}},
	}
}

// Функция для логирования в файл и на экран
func logToFileAndScreen(message string) {
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	logMessage := fmt.Sprintf("[%s] %s", timestamp, message)
	
	// Запись в файл
	log.Println(message)
	
	// Вывод на экран только если это не обычное сообщение
	if strings.Contains(strings.ToLower(message), "ошибка") {
		fmt.Println(logMessage)
	}
}

// Главное меню
func mainMenu(reader *bufio.Reader) {
	for {
		fmt.Println("\n=== МЕНЮ ===")
		fmt.Println("1. Просмотр таблицы")
		fmt.Println("2. Фильтрация")
		fmt.Println("3. Обновить запись")
		fmt.Println("4. Добавить запись")
		fmt.Println("5. Добавить запись в связанные таблицы")
		fmt.Println("0. Выход")

		fmt.Print("Выберите пункт меню: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		choice, err := strconv.Atoi(input)
		if err != nil {
			fmt.Println("Ошибка: введите цифру от 0 до 5")
			continue
		}

		switch choice {
		case 0:
			fmt.Println("Завершение программы...")
			db.Close()
			os.Exit(0)
		case 1:
			viewTable(reader)
		case 2:
			filterData(reader)
		case 3:
			updateData(reader)
		case 4:
			insertData(reader)
		case 5:
			insertRelatedData(reader)
		default:
			fmt.Println("Ошибка: выберите цифру от 0 до 5")
		}
	}
}

// Функция для выравнивания строк до заданной длины
func padRight(str string, length int) string {
	if len(str) >= length {
		return str[:length]
	}
	return str + strings.Repeat(" ", length-len(str))
}

// Пункт 1: Просмотр таблицы
func viewTable(reader *bufio.Reader) {
	for {
		fmt.Println("\n=== ВЫБОР ТАБЛИЦЫ ДЛЯ ПРОСМОТРА ===")
		for i, table := range tables {
			fmt.Printf("%d. %s\n", i+1, table.Name)
		}
		fmt.Println("0. Вернуться в меню")

		fmt.Print("Выберите таблицу: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		choice, err := strconv.Atoi(input)
		if err != nil || choice < 0 || choice > len(tables) {
			fmt.Println("Ошибка: выберите цифру от 0 до", len(tables))
			continue
		}

		if choice == 0 {
			return
		}

		tableName := tables[choice-1].Name
		query := fmt.Sprintf("SELECT * FROM %s ORDER BY id", tableName)
		
		logToFileAndScreen(fmt.Sprintf("Выполнение запроса: %s", query))
		
		rows, err := db.Query(query)
		if err != nil {
			logToFileAndScreen(fmt.Sprintf("Ошибка выполнения запроса: %v", err))
			fmt.Println("Ошибка: Не удалось выполнить запрос к таблице")
			continue
		}
		defer rows.Close()

		// Получение названий колонок
		columns, err := rows.Columns()
		if err != nil {
			logToFileAndScreen(fmt.Sprintf("Ошибка получения колонок: %v", err))
			continue
		}

		// Определяем максимальную ширину для каждой колонки
		columnWidths := make([]int, len(columns))
		for i, col := range columns {
			if len(col) > columnWidths[i] {
				columnWidths[i] = len(col)
			}
		}

		// Считываем данные для определения ширины
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		allRows := [][]string{}
		
		for rows.Next() {
			for i := range values {
				valuePtrs[i] = &values[i]
			}
			
			if err := rows.Scan(valuePtrs...); err != nil {
				logToFileAndScreen(fmt.Sprintf("Ошибка чтения строки: %v", err))
				continue
			}

			rowData := make([]string, len(columns))
			for i, val := range values {
				str := ""
				if val != nil {
					str = fmt.Sprintf("%v", val)
				}
				rowData[i] = str
				if len(str) > columnWidths[i] {
					columnWidths[i] = len(str)
				}
			}
			allRows = append(allRows, rowData)
		}

		// Если нужно переоткрыть курсор
		rows.Close()
		rows, _ = db.Query(query)
		defer rows.Close()

		// Вывод заголовков с выравниванием
		headerParts := make([]string, len(columns))
		for i, col := range columns {
			headerParts[i] = padRight(col, columnWidths[i])
		}
		fmt.Println("\n" + strings.Join(headerParts, " | "))

		// Вывод разделительной линии
		dividerParts := make([]string, len(columns))
		for i, width := range columnWidths {
			dividerParts[i] = strings.Repeat("-", width)
		}
		fmt.Println(strings.Join(dividerParts, "-+-"))

		// Вывод данных с выравниванием
		rowCount := 0
		for _, rowData := range allRows {
			rowParts := make([]string, len(rowData))
			for i, cell := range rowData {
				rowParts[i] = padRight(cell, columnWidths[i])
			}
			fmt.Println(strings.Join(rowParts, " | "))
			rowCount++
		}

		fmt.Printf("\nНайдено записей: %d\n", rowCount)
		logToFileAndScreen(fmt.Sprintf("Просмотр таблицы %s: найдено %d записей", tableName, rowCount))
		
		// Возвращаемся в главное меню после успешного выполнения
		return
	}
}

// Пункт 2: Фильтрация
func filterData(reader *bufio.Reader) {
	fmt.Print("\nВведите количество фильтров (минимум 1): ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	filterCount, err := strconv.Atoi(input)
	if err != nil || filterCount < 1 {
		fmt.Println("Ошибка: введите число больше 0")
		return
	}

	// Выбор таблицы
	tableIndex := selectTable(reader, "ВЫБОР ТАБЛИЦЫ ДЛЯ ФИЛЬТРАЦИИ")
	if tableIndex == -1 {
		return
	}

	table := tables[tableIndex]
	var conditions []string
	var values []interface{}

	for i := 0; i < filterCount; i++ {
		fmt.Printf("\n=== Фильтр %d из %d ===\n", i+1, filterCount)
		
		// Выбор колонки
		columnIndex := selectColumn(reader, table)
		if columnIndex == -1 {
			return
		}

		columnName := table.Columns[columnIndex]

		// Ввод значения для фильтрации
		fmt.Printf("Введите значение для фильтрации по '%s': ", columnName)
		value, _ := reader.ReadString('\n')
		value = strings.TrimSpace(value)

		// Проверка white list
		if !whiteListRegex.MatchString(value) {
			fmt.Println("Ошибка: значение содержит недопустимые символы")
			return
		}

		conditions = append(conditions, fmt.Sprintf("%s = $%d", columnName, i+1))
		values = append(values, value)
	}

	// Формирование и выполнение запроса
	query := fmt.Sprintf("SELECT * FROM %s WHERE %s ORDER BY id", 
		table.Name, strings.Join(conditions, " AND "))
	
	logToFileAndScreen(fmt.Sprintf("Выполнение фильтрации: %s с параметрами %v", query, values))
	
	rows, err := db.Query(query, values...)
	if err != nil {
		logToFileAndScreen(fmt.Sprintf("Ошибка выполнения фильтрации: %v", err))
		fmt.Println("Ошибка: Не удалось выполнить фильтрацию")
		return
	}
	defer rows.Close()

	// Вывод результатов
	columns, _ := rows.Columns()
	
	// Определяем ширину колонок
	columnWidths := make([]int, len(columns))
	for i, col := range columns {
		if len(col) > columnWidths[i] {
			columnWidths[i] = len(col)
		}
	}

	allRows := [][]string{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			continue
		}

		rowData := make([]string, len(columns))
		for i, val := range values {
			str := ""
			if val != nil {
				str = fmt.Sprintf("%v", val)
			}
			rowData[i] = str
			if len(str) > columnWidths[i] {
				columnWidths[i] = len(str)
			}
		}
		allRows = append(allRows, rowData)
	}

	if len(allRows) == 0 {
		fmt.Println("По заданным фильтрам записей не найдено")
		logToFileAndScreen("Фильтрация: записей не найдено")
		return
	}

	// Вывод заголовков
	headerParts := make([]string, len(columns))
	for i, col := range columns {
		headerParts[i] = padRight(col, columnWidths[i])
	}
	fmt.Println("\n" + strings.Join(headerParts, " | "))

	// Вывод разделительной линии
	dividerParts := make([]string, len(columns))
	for i, width := range columnWidths {
		dividerParts[i] = strings.Repeat("-", width)
	}
	fmt.Println(strings.Join(dividerParts, "-+-"))

	// Вывод данных
	for _, rowData := range allRows {
		rowParts := make([]string, len(rowData))
		for i, cell := range rowData {
			rowParts[i] = padRight(cell, columnWidths[i])
		}
		fmt.Println(strings.Join(rowParts, " | "))
	}

	fmt.Printf("\nНайдено записей: %d\n", len(allRows))
	logToFileAndScreen(fmt.Sprintf("Фильтрация таблицы %s: найдено %d записей", table.Name, len(allRows)))
}

// Пункт 3: Обновление данных
func updateData(reader *bufio.Reader) {
	fmt.Print("\nВведите количество данных для обновления (минимум 1): ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	updateCount, err := strconv.Atoi(input)
	if err != nil || updateCount < 1 {
		fmt.Println("Ошибка: введите число больше 0")
		return
	}

	// Выбор таблицы
	tableIndex := selectTable(reader, "ВЫБОР ТАБЛИЦЫ ДЛЯ ОБНОВЛЕНИЯ")
	if tableIndex == -1 {
		return
	}

	table := tables[tableIndex]

	// Создаем список колонок без id (id нельзя обновлять!)
	updatableColumns := make([]string, 0)
	for _, column := range table.Columns {
		if column != "id" {
			updatableColumns = append(updatableColumns, column)
		}
	}

	if len(updatableColumns) == 0 {
		fmt.Println("В таблице нет колонок для обновления")
		return
	}

	// Ввод ID для обновления
	var ids []string
	for i := 0; i < updateCount; i++ {
		fmt.Printf("Введите ID записи %d для обновления: ", i+1)
		idInput, _ := reader.ReadString('\n')
		idInput = strings.TrimSpace(idInput)
		
		if _, err := strconv.Atoi(idInput); err != nil {
			fmt.Println("Ошибка: ID должен быть числом")
			return
		}
		ids = append(ids, idInput)
	}

	// Выбор колонки для обновления (исключая id)
	fmt.Printf("\n=== ВЫБОР КОЛОНКИ ДЛЯ ОБНОВЛЕНИЯ В '%s' ===\n", table.Name)
	for i, column := range updatableColumns {
		fmt.Printf("%d. %s\n", i+1, column)
	}
	fmt.Println("0. Вернуться в меню")

	fmt.Print("Выберите колонку для обновления: ")
	columnInput, _ := reader.ReadString('\n')
	columnInput = strings.TrimSpace(columnInput)

	columnChoice, err := strconv.Atoi(columnInput)
	if err != nil || columnChoice < 0 || columnChoice > len(updatableColumns) {
		fmt.Println("Ошибка: выберите цифру от 0 до", len(updatableColumns))
		return
	}

	if columnChoice == 0 {
		return
	}

	columnName := updatableColumns[columnChoice-1]

	// Ввод нового значения
	fmt.Printf("Введите новое значение для '%s' в таблице '%s': ", columnName, table.Name)
	newValue, _ := reader.ReadString('\n')
	newValue = strings.TrimSpace(newValue)

	// Проверка white list
	if !whiteListRegex.MatchString(newValue) {
		fmt.Println("Ошибка: значение содержит недопустимые символы")
		return
	}

	// Проверка для числовых полей
	if columnName == "price" || columnName == "quantity" || columnName == "founded_year" || 
	   columnName == "category_id" || columnName == "manufacturer_id" || columnName == "component_id" {
		if _, err := strconv.Atoi(newValue); err != nil {
			fmt.Printf("Ошибка: поле '%s' должно быть числом\n", columnName)
			return
		}
	}

	// Формирование и выполнение запроса
	var query string
	var args []interface{}
	
	if updateCount == 1 {
		query = fmt.Sprintf("UPDATE %s SET %s = $1 WHERE id = $2", table.Name, columnName)
		args = []interface{}{newValue, ids[0]}
	} else {
		placeholders := make([]string, len(ids))
		args = []interface{}{newValue}
		for i, id := range ids {
			placeholders[i] = fmt.Sprintf("$%d", i+2)
			args = append(args, id)
		}
		query = fmt.Sprintf("UPDATE %s SET %s = $1 WHERE id IN (%s)", 
			table.Name, columnName, strings.Join(placeholders, ", "))
	}

	logToFileAndScreen(fmt.Sprintf("Выполнение обновления: %s с параметрами %v", query, args))
	
	result, err := db.Exec(query, args...)
	if err != nil {
		logToFileAndScreen(fmt.Sprintf("Ошибка обновления: %v", err))
		fmt.Println("Ошибка: Не удалось обновить данные")
		return
	}

	rowsAffected, _ := result.RowsAffected()
	fmt.Printf("Обновлено записей: %d\n", rowsAffected)
	logToFileAndScreen(fmt.Sprintf("Обновление таблица %s: обновлено %d записей", table.Name, rowsAffected))
}

// Пункт 4: Добавление записи
func insertData(reader *bufio.Reader) {
	fmt.Print("\nВведите количество создаваемых записей (минимум 1): ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	recordCount, err := strconv.Atoi(input)
	if err != nil || recordCount < 1 {
		fmt.Println("Ошибка: введите число больше 0")
		return
	}

	// Выбор таблицы
	tableIndex := selectTable(reader, "ВЫБОР ТАБЛИЦЫ ДЛЯ ДОБАВЛЕНИЯ")
	if tableIndex == -1 {
		return
	}

	table := tables[tableIndex]

	// Исключаем колонку id
	insertColumns := table.Columns[1:]

	for i := 0; i < recordCount; i++ {
		fmt.Printf("\n=== Ввод данных для записи %d из %d ===\n", i+1, recordCount)
		
		var values []interface{}
		for _, column := range insertColumns {
			fmt.Printf("Введите значение для '%s': ", column)
			value, _ := reader.ReadString('\n')
			value = strings.TrimSpace(value)

			// Проверка white list
			if !whiteListRegex.MatchString(value) {
				fmt.Println("Ошибка: значение содержит недопустимые символы")
				return
			}
			
			// Проверка для числовых полей
			if column == "price" || column == "quantity" || column == "founded_year" || 
			   column == "category_id" || column == "manufacturer_id" || column == "component_id" {
				if _, err := strconv.Atoi(value); err != nil {
					fmt.Printf("Ошибка: поле '%s' должно быть числом\n", column)
					return
				}
			}
			
			values = append(values, value)
		}

		// Формирование запроса
		placeholders := make([]string, len(insertColumns))
		for j := range placeholders {
			placeholders[j] = fmt.Sprintf("$%d", j+1)
		}

		query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
			table.Name,
			strings.Join(insertColumns, ", "),
			strings.Join(placeholders, ", "))

		logToFileAndScreen(fmt.Sprintf("Выполнение вставки: %s с параметрами %v", query, values))
		
		_, err := db.Exec(query, values...)
		if err != nil {
			logToFileAndScreen(fmt.Sprintf("Ошибка вставки: %v", err))
			fmt.Println("Ошибка: Не удалось добавить запись")
			return
		}

		fmt.Printf("Запись %d успешно добавлена\n", i+1)
		logToFileAndScreen(fmt.Sprintf("Добавлена запись в таблицу %s", table.Name))
	}
	
	fmt.Printf("\nВсего добавлено записей: %d\n", recordCount)
}

// Пункт 5: Добавление записи в связанные таблицы
func insertRelatedData(reader *bufio.Reader) {
	fmt.Print("\nВведите количество создаваемых записей (минимум 1): ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	recordCount, err := strconv.Atoi(input)
	if err != nil || recordCount < 1 {
		fmt.Println("Ошибка: введите число больше 0")
		return
	}

	// Выбор связанных таблиц
	fmt.Println("\n=== ВЫБОР СВЯЗАННЫХ ТАБЛИЦ ===")
	for i, relation := range relatedTables {
		fmt.Printf("%d. %s\n", i+1, relation)
	}
	fmt.Println("0. Вернуться в меню")

	fmt.Print("Выберите связанные таблицы: ")
	choiceInput, _ := reader.ReadString('\n')
	choiceInput = strings.TrimSpace(choiceInput)

	choice, err := strconv.Atoi(choiceInput)
	if err != nil || choice < 0 || choice > len(relatedTables) {
		fmt.Println("Ошибка: выберите цифру от 0 до", len(relatedTables))
		return
	}

	if choice == 0 {
		return
	}

	// Обработка выбранных связанных таблиц
	relation := relatedTables[choice-1]
	tablesInRelation := strings.Split(relation, " и ")

	if len(tablesInRelation) != 2 {
		fmt.Println("Ошибка: некорректный формат связанных таблиц")
		return
	}

	// Находим информацию о таблицах
	var table1, table2 TableInfo
	for _, t := range tables {
		if t.Name == tablesInRelation[0] {
			table1 = t
		}
		if t.Name == tablesInRelation[1] {
			table2 = t
		}
	}

	for i := 0; i < recordCount; i++ {
		fmt.Printf("\n=== Ввод данных для связанных таблиц %d из %d ===\n", i+1, recordCount)
		
		// Вставка в первую таблицу
		fmt.Printf("\n--- Данные для таблицы '%s' ---\n", table1.Name)
		insertColumns1 := table1.Columns[1:]
		var values1 []interface{}
		
		for _, column := range insertColumns1 {
			fmt.Printf("Введите значение для '%s': ", column)
			value, _ := reader.ReadString('\n')
			value = strings.TrimSpace(value)

			if !whiteListRegex.MatchString(value) {
				fmt.Println("Ошибка: значение содержит недопустимые символы")
				return
			}
			
			// Проверка числовых полей
			if column == "price" || column == "founded_year" || column == "category_id" || 
			   column == "manufacturer_id" {
				if _, err := strconv.Atoi(value); err != nil {
					fmt.Printf("Ошибка: поле '%s' должно быть числом\n", column)
					return
				}
			}
			
			values1 = append(values1, value)
		}

		placeholders1 := make([]string, len(insertColumns1))
		for j := range placeholders1 {
			placeholders1[j] = fmt.Sprintf("$%d", j+1)
		}

		query1 := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) RETURNING id",
			table1.Name,
			strings.Join(insertColumns1, ", "),
			strings.Join(placeholders1, ", "))

		logToFileAndScreen(fmt.Sprintf("Выполнение вставки в связанные таблицы: %s с параметрами %v", query1, values1))
		
		var insertedID int
		err := db.QueryRow(query1, values1...).Scan(&insertedID)
		if err != nil {
			logToFileAndScreen(fmt.Sprintf("Ошибка вставки в первую таблицу: %v", err))
			fmt.Println("Ошибка: Не удалось добавить запись в первую таблицу")
			return
		}

		fmt.Printf("✓ В таблицу '%s' добавлена запись с ID: %d\n", table1.Name, insertedID)

		// Вставка во вторую таблицу с использованием ID из первой
		fmt.Printf("\n--- Данные для таблицы '%s' ---\n", table2.Name)
		
		// Находим колонку, которая ссылается на первую таблицу
		var foreignKeyColumn string
		for _, column := range table2.Columns {
			if column == "component_id" || column == "category_id" || column == "manufacturer_id" {
				if strings.Contains(table2.Name, "stock") && table1.Name == "components" && column == "component_id" {
					foreignKeyColumn = column
					break
				} else if strings.Contains(table2.Name, "components") {
					if table1.Name == "categories" && column == "category_id" {
						foreignKeyColumn = column
						break
					} else if table1.Name == "manufacturers" && column == "manufacturer_id" {
						foreignKeyColumn = column
						break
					}
				}
			}
		}

		if foreignKeyColumn == "" {
			// Если не нашли явную связь, используем первую подходящую колонку
			for _, column := range table2.Columns {
				if column != "id" {
					foreignKeyColumn = column
					break
				}
			}
		}

		// Ввод данных для второй таблицы
		fmt.Printf("В таблицу '%s' будет добавлен внешний ключ '%s' = %d\n", table2.Name, foreignKeyColumn, insertedID)
		
		// Запрашиваем остальные данные для второй таблицы
		insertColumns2 := table2.Columns[1:] // исключаем id
		var values2 []interface{}

		for _, column := range insertColumns2 {
			if column == foreignKeyColumn {
				values2 = append(values2, insertedID)
				fmt.Printf("  Автоматически установлено: %s = %d\n", column, insertedID)
				continue
			}
			
			fmt.Printf("Введите значение для '%s': ", column)
			value, _ := reader.ReadString('\n')
			value = strings.TrimSpace(value)

			if !whiteListRegex.MatchString(value) {
				fmt.Println("Ошибка: значение содержит недопустимые символы")
				return
			}
			
			// Проверка числовых полей
			if column == "quantity" || column == "price" {
				if _, err := strconv.Atoi(value); err != nil {
					fmt.Printf("Ошибка: поле '%s' должно быть числом\n", column)
					return
				}
			}
			
			values2 = append(values2, value)
		}

		placeholders2 := make([]string, len(insertColumns2))
		for j := range placeholders2 {
			placeholders2[j] = fmt.Sprintf("$%d", j+1)
		}

		query2 := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
			table2.Name,
			strings.Join(insertColumns2, ", "),
			strings.Join(placeholders2, ", "))

		logToFileAndScreen(fmt.Sprintf("Выполнение вставки во вторую таблицу: %s с параметрами %v", query2, values2))
		
		_, err = db.Exec(query2, values2...)
		if err != nil {
			logToFileAndScreen(fmt.Sprintf("Ошибка вставки во вторую таблицу: %v", err))
			fmt.Println("Ошибка: Не удалось добавить запись во вторую таблицу")
			return
		}

		fmt.Printf("✓ В таблицу '%s' успешно добавлена запись\n", table2.Name)
		logToFileAndScreen(fmt.Sprintf("Добавлены записи в связанные таблицы %s", relation))
	}
	
	fmt.Printf("\nВсего добавлено связанных записей: %d\n", recordCount)
}

// Вспомогательная функция для выбора таблицы
func selectTable(reader *bufio.Reader, title string) int {
	fmt.Printf("\n=== %s ===\n", title)
	for i, table := range tables {
		fmt.Printf("%d. %s\n", i+1, table.Name)
	}
	fmt.Println("0. Вернуться в меню")

	fmt.Print("Выберите таблицу: ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	choice, err := strconv.Atoi(input)
	if err != nil || choice < 0 || choice > len(tables) {
		fmt.Println("Ошибка: выберите цифру от 0 до", len(tables))
		return -1
	}

	if choice == 0 {
		return -1
	}

	return choice - 1
}

// Вспомогательная функция для выбора колонки
func selectColumn(reader *bufio.Reader, table TableInfo) int {
	fmt.Printf("\n=== ВЫБОР КОЛОНКИ В ТАБЛИЦЕ '%s' ===\n", table.Name)
	for i, column := range table.Columns {
		fmt.Printf("%d. %s\n", i+1, column)
	}
	fmt.Println("0. Вернуться в меню")

	fmt.Print("Выберите колонку: ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	choice, err := strconv.Atoi(input)
	if err != nil || choice < 0 || choice > len(table.Columns) {
		fmt.Println("Ошибка: выберите цифру от 0 до", len(table.Columns))
		return -1
	}

	if choice == 0 {
		return -1
	}

	return choice - 1
}
