package main

import (
	"encoding/csv"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"sync"
	"syscall/js"
	"time"
)

const RickRollEmbed = `<iframe width="560" height="315" src="https://www.youtube.com/embed/dQw4w9WgXcQ?autoplay=1" title="YouTube video player" frameborder="0" allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture" allowfullscreen></iframe>`

type player struct {
	name     js.Value
	database js.Value
	distance int
	avatar   *playerAvatar
}

func (p *player) Name() string {
	return p.name.Get("value").String()
}

func (p *player) Database() string {
	return p.database.Get("value").String()
}

type playerAvatar struct {
	x      int
	y      int
	width  int
	height int
	image  js.Value
}

type race struct {
	max                int
	distance_per_pixel int
	steps              []step
}

type step struct {
	created_at time.Time
	database   string
	distance   int
}

func FetchRaceData() race {
	race := race{}
	fmt.Printf("Fetching data...\n")
	res, err := http.DefaultClient.Get("http://localhost:8888/racedata.csv")
	if err != nil {
		fmt.Printf("Error Fetching Racedata :( %+v\n", err)
	}
	defer res.Body.Close()
	csvrd := csv.NewReader(res.Body)
	summary := make(map[string]int, 0)
	for {
		cols, err := csvrd.Read()

		if err == io.EOF {
			break
		}
		if err != nil {
			fmt.Printf("Error with CSV: %+v", err)
			break
		}

		created_at, _ := time.Parse("2006-01-02 15:04:05", cols[0])
		database := cols[1]
		distance, _ := strconv.Atoi(cols[2])
		race.steps = append(race.steps, step{created_at, database, distance})
		summary[database] += distance
	}

	for _, v := range summary {
		if v > race.max {
			race.max = v
		}
	}
	race.distance_per_pixel = race.max / 1280

	fmt.Printf("Content-Length: %d\n", res.ContentLength)
	fmt.Printf("Records: %d\n", len(race.steps))
	fmt.Printf("Max Value: %d\n", race.max)
	fmt.Printf("Max Value/Div: %d\n", race.distance_per_pixel)
	return race
}

type AssetCatalog struct {
	Ship1      js.Value
	Ship2      js.Value
	Ship3      js.Value
	Background js.Value
}

func LoadAssetCatalog() AssetCatalog {
	cat := AssetCatalog{}

	cat.Ship1 = js.Global().Get("Image").New()
	cat.Ship1.Set("src", "https://raw.githubusercontent.com/mhat/hackday202205/main/img/Spaceship3.png")

	cat.Ship2 = js.Global().Get("Image").New()
	cat.Ship2.Set("src", "https://raw.githubusercontent.com/mhat/hackday202205/main/img/Spaceship2.png")

	cat.Ship3 = js.Global().Get("Image").New()
	cat.Ship3.Set("src", "https://raw.githubusercontent.com/mhat/hackday202205/main/img/Spaceship1.png")

	cat.Background = js.Global().Get("Image").New()
	cat.Background.Set("src", "https://raw.githubusercontent.com/mhat/hackday202205/main/img/Background.png")

	time.Sleep(1 * time.Second)
	return cat
}

func LoadPlayers(cat AssetCatalog) []*player {
	players := make([]*player, 0)
	players = append(players, &player{
		name:     js.Global().Get(fmt.Sprintf("p%dname", 1)),
		database: js.Global().Get(fmt.Sprintf("p%ddatabase", 1)),
		avatar:   &playerAvatar{x: 0, y: 70, width: 150, height: 150, image: cat.Ship1},
	})

	players = append(players, &player{
		name:     js.Global().Get(fmt.Sprintf("p%dname", 2)),
		database: js.Global().Get(fmt.Sprintf("p%ddatabase", 2)),
		avatar:   &playerAvatar{x: 0, y: 160, width: 150, height: 150, image: cat.Ship2},
	})

	players = append(players, &player{
		name:     js.Global().Get(fmt.Sprintf("p%dname", 3)),
		database: js.Global().Get(fmt.Sprintf("p%ddatabase", 3)),
		avatar:   &playerAvatar{x: 0, y: 260, width: 150, height: 150, image: cat.Ship3},
	})
	return players
}

func main() {
	catalog := LoadAssetCatalog()
	players := LoadPlayers(catalog)
	race := FetchRaceData()

	DrawScene(catalog, players)

	runRaceC := make(chan bool)
	raceButton := js.Global().Get("raceButton")
	raceButton.Call("addEventListener", "click", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		runRaceC <- true
		return nil
	}))

	for {
		<-runRaceC
		go RunRace(catalog, race, players)
	}
}

func DrawScene(cat AssetCatalog, players []*player) {
	canvas := js.Global().Get("track")
	canctx := canvas.Call("getContext", "2d")

	canctx.Call("drawImage", cat.Background, 0, 0)
	for _, player := range players {
		avatar := player.avatar
		canctx.Set("fillStyle", "orange")
		canctx.Call("fillRect", 0, avatar.y+50, avatar.x, 10)
		canctx.Call("drawImage", avatar.image, avatar.x, avatar.y)
	}
}

func RunRace(cat AssetCatalog, race race, players []*player) {
	wg := &sync.WaitGroup{}
	wg.Add(1)
	stepIdx := 0
	stepId := js.Global().Call("setInterval", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		for j := 0; j <= 100; j++ {
			if stepIdx >= len(race.steps)-1 {
				fmt.Printf("Stepper Complete after %d out of %d!\n", stepIdx, len(race.steps)-1)
				defer wg.Done()
				return nil
			}

			stepIdx += 1
			step := race.steps[stepIdx]
			for _, p := range players {
				if step.database == p.Database() {
					a := p.avatar
					if a.x <= 1280-150 {
						p.distance += step.distance
						a.x = p.distance / race.distance_per_pixel
					}
				}
			}
		}
		return nil
	}), js.ValueOf(10))

	// Animation Looper
	var drawFn js.Func
	drawFn = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		DrawScene(cat, players)
		js.Global().Call("requestAnimationFrame", drawFn)
		return nil
	})
	js.Global().Call("requestAnimationFrame", drawFn)

	// Cleanup??
	fmt.Printf("stepId: %s\n", stepId)
	wg.Wait()
	fmt.Printf("Clearing Intervals\n")
	js.Global().Call("clearInterval", stepId)

	var winningPlayer player
	for _, p := range players {
		if winningPlayer.distance < p.distance {
			winningPlayer = *p
		}
		fmt.Printf("Dump: %+v\n", p)
		fmt.Printf("Dump-Avatar: %+v\n", *p.avatar)
	}
	fmt.Printf("Race Complete! Dialog Up\n")
	DisplayWinnerDialog(winningPlayer)
	// players = LoadPlayers(cat)

}

func DisplayWinnerDialog(p player) {
	js.Global().Get("rickroll").Set("innerHTML", RickRollEmbed)
	js.Global().Get("winnerMessage").Set("innerHTML",
		fmt.Sprintf("<h2>%s is the winner with %s!<h2>", p.Name(), p.Database()))

	dialogStyle := js.Global().Get("winnerDialog").Get("style")
	canvasStyle := js.Global().Get("track").Get("style")

	fmt.Printf("top=[%+v], left=[%+v]\n", canvasStyle.Get("top"), canvasStyle.Get("left"))
	dialogStyle.Set("top", "332px")
	dialogStyle.Set("left", "380px")
	dialogStyle.Set("border", "3px solid green")
	dialogStyle.Set("background", "green")
	dialogStyle.Set("text-align", "center")
	dialogStyle.Set("display", canvasStyle.Get("display"))
}
