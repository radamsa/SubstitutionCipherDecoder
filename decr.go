package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"unicode/utf8"
)

type mask_t []uint8

type word_t struct {
	word   string
	weight int
	mask   mask_t
}

type symbol_t struct {
	ch    rune
	fixed bool
}

type alphabet_t map[rune]symbol_t
type words_t []word_t
type dictionary_t map[int]words_t

var dictionary dictionary_t
var alphabet alphabet_t

// const DICTIONARY_FILE string = "litw-win-utf8.txt"
// const SOURCE_FILE string = "r02-2-utf8.txt"
// const TARGET_FILE string = "r02-2-utf8-decoded.txt"

func main() {
	if len(os.Args) < 4 {
		fmt.Println("USAGE:")
		fmt.Printf("%v <dictionary> <source> <target>", os.Args[0])
		return
	}

	DICTIONARY_FILE := os.Args[1]
	SOURCE_FILE := os.Args[2]
	TARGET_FILE := os.Args[3]

	// Заполним алфавит
	preAlphabet := []rune{
		'О', 'Е', 'А', 'И', 'Н', 'Т', 'С', 'Р', 'В', 'Л', 'К', 'М', 'Д', 'П', 'У', 'Я',
		'Ы', 'З', 'Б', 'Ь', 'Ъ', 'Г', 'Ч', 'Й', 'Х', 'Ж', 'Ш', 'Ю', 'Ц', 'Щ', 'Э', 'Ф'}
	unusedAlphas := map[rune]int{
		'О': 0, 'Е': 0, 'А': 0, 'И': 0, 'Н': 0, 'Т': 0, 'С': 0, 'Р': 0, 'В': 0, 'Л': 0, 'К': 0, 'М': 0, 'Д': 0, 'П': 0, 'У': 0, 'Я': 0,
		'Ы': 0, 'З': 0, 'Б': 0, 'Ь': 0, 'Ъ': 0, 'Г': 0, 'Ч': 0, 'Й': 0, 'Х': 0, 'Ж': 0, 'Ш': 0, 'Ю': 0, 'Ц': 0, 'Щ': 0, 'Э': 0, 'Ф': 0}

	// Читаем слова из файла и формируем словарь в памяти
	dictionary, err := readDictionary(DICTIONARY_FILE)
	if err != nil {
		fmt.Println("Ошибка загрузки словаря: ", err)
		return
	}

	fmt.Println("Словарь успешно загружен. Записей верхнего уровня: ", len(dictionary))

	text, err := readText(SOURCE_FILE)
	if err != nil {
		fmt.Println("Ошибка загрузки зашифрованного текста: ", err)
		return
	}

	fmt.Println("Загружен зашифрованный текст. Слов в тексте: ", len(text))

	// Посчитаем частоту символов в зашифрованном тексте
	encryptedAlphas := countChars(text)

	// Проинициализируем начальное состояние алфавита в соответствии с полученной частотой символов в тексте
	alphabet = make(alphabet_t)
	i := 0
	for _, ch := range sortedKeys(encryptedAlphas) {
		alphabet[ch] = symbol_t{preAlphabet[i], false}
		delete(unusedAlphas, ch)
		i++
	}
	if len(unusedAlphas) > 0 {
		fmt.Println("Следующие символы не встречаются в тексте: ")
		for ch, _ := range unusedAlphas {
			fmt.Printf("'%v', ", string(ch))
			alphabet[ch] = symbol_t{preAlphabet[i], false}
			i++
		}
		fmt.Println()
	}
	if i != len(preAlphabet) {
		fmt.Errorf("Ошибка заполнения словаря. Осталось неиспольщованных букв: %v", len(preAlphabet)-i)
		return
	}

	fmt.Println("Выполняется декодирование...")
	targetAlphabet := Decode(text, dictionary, alphabet)

	fmt.Println("Алфавит:")
	PrintAlphabet(alphabet)

	err = DecodeFile(SOURCE_FILE, TARGET_FILE, targetAlphabet)
	if err != nil {
		fmt.Println("В процессе декодирования файла возникла ошибка: ", err)
	}
}

func Decode(text []string, dictionary dictionary_t, alphabet alphabet_t) alphabet_t {
	// Сформируем массив зашифрованных слов, разделенных по их размеру (как в словаре)
	encryptedWords := make(map[int][]string)
	for _, word := range text {
		l := utf8.RuneCountInString(word)
		encryptedWords[l] = append(encryptedWords[l], word)
	}

	// Для каждого закодированного слова найдем все "похожие" слова
	// В процессе обработки найдем слова, у которых только одно похожее слово из словаря.
	similarWords := make(map[string]words_t)
	fmt.Println("Слова, у которых только одно совпадение со соварем:")
	var iterationOrder []int
	for l, words := range encryptedWords {
		iterationOrder = append(iterationOrder, l)
		for _, word := range words {
			similarWords[word] = getSimilarWords(dictionary[l], word, alphabet)
			if len(similarWords[word]) == 1 {
				fmt.Printf("%v == %v\n", word, similarWords[word][0].word)
				alphabet = fixAlphabetBasedOnWord(word, similarWords[word][0].word, alphabet)
			}
		}
	}

	fmt.Println("Iteration order was: ", iterationOrder)

	return alphabet
}

func readDictionary(path string) (dictionary_t, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	lines := make(dictionary_t)
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		str := strings.TrimSpace(strings.ToUpper(scanner.Text()))
		word := strings.Split(str, " ")

		if len(word) != 2 {
			fmt.Printf("Обнаружена некорректная запись в словаре: '%v'\n", str)
		}

		length := utf8.RuneCountInString(word[1])
		weight, err := strconv.Atoi(word[0])
		if err != nil {
			return nil, err
		}

		var record word_t
		record.word = word[1]
		record.weight = weight
		record.mask = getWordMask(record.word)
		lines[length] = append(lines[length], record)
	}

	return lines, scanner.Err()
}

func getWordMask(word string) mask_t {
	m := make(map[rune]uint8)
	var result mask_t
	var i uint8
	i = 0
	for _, ch := range word {
		_, ok := m[ch]
		if !ok {
			m[ch] = i
			i++
		}
		result = append(result, m[ch])
	}

	return result
}

func readText(path string) ([]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	scanner.Split(bufio.ScanWords)
	for scanner.Scan() {
		str := strings.TrimSpace(strings.ToUpper(scanner.Text()))
		var buffer bytes.Buffer
		hasAlpha := false
		for _, ch := range str {
			if ch >= 'А' && ch <= 'Я' {
				buffer.WriteString(string(ch))
				hasAlpha = true
			}
			// if ch == '-' {
			// 	buffer.WriteString(string(ch))
			// }
		}
		if hasAlpha {
			// Есть некоторая вероятность, что символы тире встретятся в начале или конце строки.
			// Уберем их.
			lines = append(lines, strings.Trim(buffer.String(), "-"))
		}
	}

	return lines, scanner.Err()
}

func decodeWord(word string, alphabet alphabet_t) string {
	var buffer bytes.Buffer

	for _, ch := range word {
		buffer.WriteString(string(alphabet[ch].ch))
	}

	return buffer.String()
}

// Раскодирует текст используя словарь
func DecodeFile(sourcePath, targetPath string, alphabet alphabet_t) error {
	sourceFile, err := os.Open(sourcePath)
	if err != nil {
		return err
	}
	defer sourceFile.Close()
	reader := bufio.NewReader(sourceFile)

	targetFile, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer targetFile.Close()
	writer := bufio.NewWriter(targetFile)
	defer writer.Flush()

	for {
		ch, _, err := reader.ReadRune()
		if err != nil {
			if err == io.EOF {
				return nil
			}
			return err
		}
		if ch >= 'А' && ch <= 'Я' {
			_, err = writer.WriteRune(alphabet[ch].ch)
		} else {
			_, err = writer.WriteRune(ch)
		}
		if err != nil {
			return err
		}
	}

	return nil
}

func decodeText(words []string, alphabet alphabet_t) []string {
	var result []string

	for _, w := range words {
		result = append(result, decodeWord(w, alphabet))
	}

	return result
}

func countChars(words []string) map[rune]int {
	m := make(map[rune]int)
	for _, word := range words {
		for _, ch := range word {
			if ch != '-' { // В тексте может встречаеться тире, но оно не кодируется, поэтому считать его не будем
				_, ok := m[ch]
				if ok {
					m[ch]++
				} else {
					m[ch] = 1
				}
			}
		}
	}

	return m
}

func compareMask(m1, m2 mask_t) int {
	if len(m1) < len(m2) {
		return -1
	}
	if len(m1) > len(m2) {
		return 1
	}

	for i := 0; i < len(m1); i++ {
		if m1[i] < m2[i] {
			return -1
		} else if m1[i] > m2[i] {
			return 1
		}
	}

	return 0
}

// Сравнивает закодированное слово со словарным учитывая зафиксированные буквы словаря
// w1 - зашифрованное слово
// w2 - словарное слово
// alphabet - словарь
// При сравнении учитываются только те буквы, которые зафиксированы в словаре
func compareSimilarWords(w1, w2 string, alphabet alphabet_t) int {
	r1 := []rune(w1)
	r2 := []rune(w2)
	l1 := len(r1)
	l2 := len(r2)

	if l1 < l2 {
		return -1
	}
	if l1 > l2 {
		return 1
	}

	for i := 0; i < l1; i++ {
		if alphabet[r1[i]].fixed {
			if alphabet[r1[i]].ch < r2[i] {
				return -1
			} else if alphabet[r1[i]].ch > r2[i] {
				return 1
			}
		}
	}

	return 0
}

func getSimilarWords(words words_t, word string, alphabet alphabet_t) words_t {
	mask := getWordMask(word)
	result := make(words_t, 0)
	for _, w := range words {
		// Сначала сравним маски слов
		if compareMask(w.mask, mask) == 0 {
			// Для слов, чьи маски совпадают сравним совпадение зафиксированных букв алфавита
			if compareSimilarWords(word, w.word, alphabet) == 0 {
				result = append(result, w)
			}
		}
	}

	return result
}

func countMatchedWords(words []string, dictionary dictionary_t) int {
	count := 0

	// Ищем каждое слово в словаре
	for _, word := range words {
		l := utf8.RuneCountInString(word)
		for _, w := range dictionary[l] {
			if word == w.word {
				count++
				break
			}
		}
	}

	return count
}

func PrintAlphabet(alphabet alphabet_t) {
	for key, val := range alphabet {
		fmt.Printf("'%v': '%v' - %v\n", string(key), string(val.ch), val.fixed)
	}
}

func fixAlphabetRune(from, to rune, alphabet alphabet_t) alphabet_t {
	result := alphabet
	for key, sym := range result {
		if sym.ch == to {
			result[from], result[key] = result[key], result[from]
			tmp := result[from]
			tmp.fixed = true
			result[from] = tmp

			break
		}
	}

	return result
}

func fixAlphabetBasedOnWord(encryptedWord string, dicrionaryWord string, alphabeth alphabet_t) alphabet_t {
	er := []rune(encryptedWord)
	dr := []rune(dicrionaryWord)

	if len(er) != len(dr) {
		panic("Для фиксации алфавита использованы слова разной длины '" + encryptedWord + "' и '" + dicrionaryWord + "'")
	}

	result := alphabet
	for i := 0; i < len(er); i++ {
		result = fixAlphabetRune(er[i], dr[i], result)
	}

	return result
}
