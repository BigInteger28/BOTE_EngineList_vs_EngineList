package main

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// **elementsDepth** definieert de transformaties van elementen op verschillende dieptes (1-9)
var elementsDepth = map[byte]map[int]byte{
	'W': {1: 'L', 2: 'A', 3: 'V', 4: 'W', 5: 'L', 6: 'A', 7: 'V', 8: 'W', 9: 'L'},
	'V': {1: 'W', 2: 'L', 3: 'A', 4: 'V', 5: 'W', 6: 'L', 7: 'A', 8: 'V', 9: 'W'},
	'A': {1: 'V', 2: 'W', 3: 'L', 4: 'A', 5: 'V', 6: 'W', 7: 'L', 8: 'A', 9: 'V'},
	'L': {1: 'A', 2: 'V', 3: 'W', 4: 'L', 5: 'A', 6: 'V', 7: 'W', 8: 'L', 9: 'A'},
}

// **moveWins** definieert bitwise wie wint (1 = move1 wint, 2 = move2 wint, 0 = gelijk)
var moveWins = [5][5]uint8{
	{0, 1, 0, 2, 0}, // W vs W,V,A,L,D
	{2, 0, 1, 0, 0}, // V
	{0, 2, 0, 1, 0}, // A
	{1, 0, 2, 0, 0}, // L
	{0, 0, 0, 0, 0}, // D
}

// **moveToIndex** converteert een move naar een index
var moveToIndex = map[byte]int{
	'W': 0,
	'V': 1,
	'A': 2,
	'L': 3,
	'D': 4,
}

// **depthToElement** converteert een diepte naar een element
var depthToElement = [10]byte{' ', 'W', 'V', 'A', 'L', 'D', 'W', 'V', 'A', 'L'}

// **Player** houdt de staat van een speler bij
type Player struct {
	available [5]int // W, V, A, L, D
	moves     [13]byte
	moveCount int
}

// **engineResult** houdt een engine, zijn totaalscore, of het nooit verliest en het aantal wins bij
type engineResult struct {
	engine     string
	score      int
	neverLoses bool
	wins       int
}

// Globale variabelen voor voortgang
var progressComparisons int64
var totalComparisons int64
var updateInterval int64 = 10000000 // Update na elke 10M matches
var startTime time.Time

// **getElementFromCode** haalt een element op basis van de diepte (1-9)
func getElementFromCode(depth int) byte {
	if depth < 1 || depth > 9 {
		return 0
	}
	return depthToElement[depth]
}

// **getElementByDepth** berekent het volgende element
func getElementByDepth(prevElement byte, depth int) byte {
	if depth == 5 || depth == 9 {
		return 'D'
	}
	if prevElement == 0 {
		return 0
	}
	if prevElement == 'D' {
		prevElement = 'L'
	}
	next, ok := elementsDepth[prevElement][depth]
	if !ok {
		return 0
	}
	return next
}

// **chooseAvailableElement** kiest een beschikbaar element
func chooseAvailableElement(target byte, available *[5]int) byte {
	targetIdx := moveToIndex[target]
	if available[targetIdx] > 0 {
		return target
	}
	current := target
	for i := 0; i < 5; i++ {
		current = elementsDepth[current][1]
		currentIdx := moveToIndex[current]
		if available[currentIdx] > 0 {
			return current
		}
	}
	if available[4] > 0 { // D
		return 'D'
	}
	return 0
}

// **getLastElement** bepaalt het resterende element voor de 13e zet
func getLastElement(available *[5]int) byte {
	candidates := [5]byte{'W', 'V', 'A', 'L', 'D'}
	for _, c := range candidates {
		if available[moveToIndex[c]] > 0 {
			return c
		}
	}
	return 0
}

// **determineWinner** bepaalt de winnaar
func determineWinner(move1, move2 byte) int {
	move1Idx, ok1 := moveToIndex[move1]
	move2Idx, ok2 := moveToIndex[move2]
	if !ok1 || !ok2 {
		return 0
	}
	return int(moveWins[move1Idx][move2Idx])
}

// **simulateDepthGame** simuleert een spel met diepte-gebaseerde codes (1-9)
func simulateDepthGame(engine1, engine2 string) (p1Score, p2Score int) {
	if len(engine1) != 12 || len(engine2) != 12 {
		return -1, -1
	}
	var p1, p2 Player
	p1.available = [5]int{3, 3, 3, 3, 1}
	p2.available = [5]int{3, 3, 3, 3, 1}

	for i := 0; i < 12; i++ {
		depth1 := int(engine1[i] - '0')
		depth2 := int(engine2[i] - '0')
		var move1, move2 byte

		if i == 0 {
			move1 = chooseAvailableElement(getElementFromCode(depth1), &p1.available)
			move2 = chooseAvailableElement(getElementFromCode(depth2), &p2.available)
		} else {
			move1 = chooseAvailableElement(getElementByDepth(p2.moves[i-1], depth1), &p1.available)
			move2 = chooseAvailableElement(getElementByDepth(p1.moves[i-1], depth2), &p2.available)
		}

		if move1 == 0 || move2 == 0 {
			return -1, -1
		}

		p1.available[moveToIndex[move1]]--
		p1.moves[p1.moveCount] = move1
		p1.moveCount++

		p2.available[moveToIndex[move2]]--
		p2.moves[p2.moveCount] = move2
		p2.moveCount++

		winner := determineWinner(move1, move2)
		if winner == 1 {
			p1Score++
		} else if winner == 2 {
			p2Score++
		}
	}

	// Laatste zet
	move1 := getLastElement(&p1.available)
	move2 := getLastElement(&p2.available)

	if move1 != 0 {
		p1.available[moveToIndex[move1]]--
		p1.moves[p1.moveCount] = move1
	}
	if move2 != 0 {
		p2.available[moveToIndex[move2]]--
		p2.moves[p2.moveCount] = move2
	}

	winner := determineWinner(move1, move2)
	if winner == 1 {
		p1Score++
	} else if winner == 2 {
		p2Score++
	}

	return p1Score, p2Score
}

// **simulateFixedGame** simuleert een spel met vaste zetten
func simulateFixedGame(engine1, engine2 string) (p1Score, p2Score int) {
	if len(engine1) != 13 || len(engine2) != 13 {
		return -1, -1
	}
	for i := 0; i < 13; i++ {
		move1, move2 := engine1[i], engine2[i]
		validMoves := map[byte]bool{'W': true, 'V': true, 'A': true, 'L': true, 'D': true}
		if !validMoves[move1] || !validMoves[move2] {
			return -1, -1
		}
		winner := determineWinner(move1, move2)
		if winner == 1 {
			p1Score++
		} else if winner == 2 {
			p2Score++
		}
	}
	return p1Score, p2Score
}

// **simulateDepthGameToMoves** genereert zetten voor een diepte-engine
func simulateDepthGameToMoves(engine string, opponent string) (moves [13]byte) {
	if len(engine) != 12 || len(opponent) != 13 {
		return
	}
	p := Player{
		available: [5]int{3, 3, 3, 3, 1},
	}
	for i := 0; i < 12; i++ {
		depth := int(engine[i] - '0')
		var move byte
		if i == 0 {
			move = chooseAvailableElement(getElementFromCode(depth), &p.available)
		} else {
			move = chooseAvailableElement(getElementByDepth(opponent[i-1], depth), &p.available)
		}
		if move != 0 {
			p.available[moveToIndex[move]]--
			moves[p.moveCount] = move
			p.moveCount++
		}
	}
	move := getLastElement(&p.available)
	if move != 0 {
		p.available[moveToIndex[move]]--
		moves[p.moveCount] = move
	} else {
		moves[p.moveCount] = 'W'
	}
	return moves
}

// **simulateGame** simuleert een spel tussen twee engines
func simulateGame(engine1, engine2 string) (p1Score, p2Score int) {
	if len(engine1) == 12 && len(engine2) == 12 {
		return simulateDepthGame(engine1, engine2)
	} else if len(engine1) == 13 && len(engine2) == 13 {
		return simulateFixedGame(engine1, engine2)
	} else if len(engine1) == 12 && len(engine2) == 13 {
		p1Moves := simulateDepthGameToMoves(engine1, engine2)
		if p1Moves[12] == 0 {
			return -1, -1
		}
		return simulateFixedGame(string(p1Moves[:]), engine2)
	} else if len(engine1) == 13 && len(engine2) == 12 {
		p2Moves := simulateDepthGameToMoves(engine2, engine1)
		if p2Moves[12] == 0 {
			return -1, -1
		}
		return simulateFixedGame(engine1, string(p2Moves[:]))
	} else {
		return -1, -1
	}
}

// **evaluateBatch** evalueert een batch van engines tegen offer engines
func evaluateBatch(batch []string, offerEngines []string, resultChan chan<- engineResult, progressComparisons *int64) {
	var printMu sync.Mutex
	for _, engine := range batch {
		totalScore := 0
		neverLoses := true
		wins := 0
		for _, offerEngine := range offerEngines {
			p1Score, p2Score := simulateGame(engine, offerEngine)
			if p1Score == -1 || p2Score == -1 {
				continue
			}
			diff := p1Score - p2Score
			if p1Score > p2Score {
				totalScore += diff + 10
				wins++
			} else if p1Score < p2Score {
				totalScore += diff - 10
				neverLoses = false
			} else {
				totalScore += p1Score
			}

			atomic.AddInt64(progressComparisons, 1)
			p := atomic.LoadInt64(progressComparisons)
			if p%updateInterval == 0 {
				printMu.Lock()
				elapsed := time.Since(startTime).Seconds()
				if elapsed > 0 {
					speed := float64(p) / elapsed / 1000
					fmt.Printf("Voortgang: %d / %d matches (%.2f%%), Snelheid: %.1f k matches/s\n",
						p, totalComparisons, float64(p)/float64(totalComparisons)*100, speed)
				} else {
					fmt.Printf("Voortgang: %d / %d matches (%.2f%%)\n",
						p, totalComparisons, float64(p)/float64(totalComparisons)*100)
				}
				printMu.Unlock()
			}
		}
		resultChan <- engineResult{engine: engine, score: totalScore, neverLoses: neverLoses, wins: wins}
	}
}

// **parseEngineCode** haalt de engine code uit invoer
func parseEngineCode(input string) string {
	parts := strings.Split(input, ":")
	engine := strings.TrimSpace(parts[len(parts)-1])

	// Diepte-engine (12 cijfers 1-9)
	if len(engine) >= 12 {
		potential := engine[:12]
		valid := true
		for _, ch := range potential {
			if ch < '1' || ch > '9' {
				valid = false
				break
			}
		}
		if valid {
			return potential
		}
	}

	// Vaste engine (13 tekens WVALD)
	if len(engine) >= 13 {
		potential := engine[:13]
		valid := true
		for _, ch := range potential {
			if !strings.ContainsRune("WVALD", ch) {
				valid = false
				break
			}
		}
		if valid {
			return potential
		}
	}

	return ""
}

func main() {
	for {
		// Lees input engines
		var inputEngines []string
		fmt.Println("Voer input engine codes in die beoordeeld moeten worden (één per regel, '.' om te stoppen):")
		scanner := bufio.NewScanner(os.Stdin)
		for scanner.Scan() {
			input := strings.TrimSpace(scanner.Text())
			if input == "." || input == "" {
				break
			}
			engine := parseEngineCode(input)
			if engine != "" {
				inputEngines = append(inputEngines, engine)
			} else {
				fmt.Printf("Ongeldige engine code '%s'.\n", input)
			}
		}
		if len(inputEngines) == 0 {
			fmt.Println("Geen input engine codes ingevoerd. Gestopt.")
			break
		}

		// Lees offer engines
		var offerEngines []string
		fmt.Println("Voer offer engine codes in waartegen gespeeld zal worden (één per regel, '.' om te stoppen):")
		for scanner.Scan() {
			input := strings.TrimSpace(scanner.Text())
			if input == "." || input == "" {
				break
			}
			engine := parseEngineCode(input)
			if engine != "" {
				offerEngines = append(offerEngines, engine)
			} else {
				fmt.Printf("Ongeldige engine code '%s'.\n", input)
			}
		}
		if len(offerEngines) == 0 {
			fmt.Println("Geen offer engine codes ingevoerd. Gestopt.")
			break
		}

		// Vraag sorteeroptie
		fmt.Println("Kies een sorteeroptie:")
		fmt.Println("1. Sorteer de volledige lijst op basis van score")
		fmt.Println("2. Output engines die nooit verliezen (gelijk of win) en sorteer op score")
		fmt.Println("3. Output engines die nooit verliezen en een minimumaantal overwinningen hebben, gesorteerd op score")
		var option int
		_, _ = fmt.Scanln(&option)

		var minWins int
		if option == 3 {
			fmt.Println("Voer het minimumaantal overwinningen in:")
			_, _ = fmt.Scanln(&minWins)
		}

		totalComparisons = int64(len(inputEngines)) * int64(len(offerEngines))
		resultChan := make(chan engineResult, len(inputEngines))
		var wg sync.WaitGroup
		progressComparisons = 0
		startTime = time.Now()

		// Parallelle verwerking
		numThreads := 16
		enginesPerThread := (len(inputEngines) + numThreads - 1) / numThreads
		for i := 0; i < numThreads; i++ {
			start := i * enginesPerThread
			end := start + enginesPerThread
			if end > len(inputEngines) {
				end = len(inputEngines)
			}
			if start < end {
				wg.Add(1)
				go func(threadStart, threadEnd int) {
					defer wg.Done()
					batch := inputEngines[threadStart:threadEnd]
					evaluateBatch(batch, offerEngines, resultChan, &progressComparisons)
				}(start, end)
			}
		}

		go func() {
			wg.Wait()
			close(resultChan)
		}()

		// Verzamel resultaten
		var results []engineResult
		for result := range resultChan {
			results = append(results, result)
		}

		// Filter en sorteer
		var filteredResults []engineResult
		switch option {
		case 1:
			filteredResults = results
		case 2:
			for _, r := range results {
				if r.neverLoses {
					filteredResults = append(filteredResults, r)
				}
			}
		case 3:
			for _, r := range results {
				if r.neverLoses && r.wins >= minWins {
					filteredResults = append(filteredResults, r)
				}
			}
		}

		// Sorteer op score (aflopend)
		sort.Slice(filteredResults, func(i, j int) bool {
			return filteredResults[i].score > filteredResults[j].score
		})

		// Schrijf naar bestand
		file, _ := os.Create("sorted_engines.txt")
		defer file.Close()

		for _, r := range filteredResults {
			file.WriteString(fmt.Sprintf("%s (score: %d, never loses: %t, wins: %d)\n",
				r.engine, r.score, r.neverLoses, r.wins))
		}

		fmt.Printf("Gesorteerde engines opgeslagen in 'sorted_engines.txt' uit %d matches.\n", totalComparisons)
	}
}
