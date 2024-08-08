package tests

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"net/http"
	"order-event-processor/internal/config"
	"order-event-processor/internal/model"
	"order-event-processor/internal/storage/postgresql"
	"order-event-processor/test/scenario"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
	"time"
)

var url = "http://localhost:8080"

func ExecuteScenario(t *testing.T, path string, timeoutBeforeListeningStream time.Duration, expectedEventIdsInOrder []string) {
	clearDb(t)
	files, err := ReadScenarioDir(path)
	if err != nil {
		t.Fatal(err)
	}
	orderId := getOrderId(files[0].Content, t)
	go func() {
		time.Sleep(timeoutBeforeListeningStream * time.Millisecond)
		resp, err := http.Get(fmt.Sprintf("%s/orders/%s/events", url, orderId))
		if err != nil {
			fmt.Println("Error connecting to SSE stream:", err)
			t.Error(err)
		}
		defer resp.Body.Close()

		reader := bufio.NewReader(resp.Body)

		fmt.Println("Connected to SSE stream. Listening for events...")

		index := 0
		for {
			line, err := reader.ReadString('\n')
			if err != nil {
				t.Error(err)
				break
			}

			line = strings.TrimSpace(line)

			if len(line) == 0 {
				continue
			}
			if strings.Index(line, "data: ") != 0 {
				t.Errorf(fmt.Sprintf("Invalid SSE stream"))
			}

			event := model.OrderEvent{}
			data := line[6:]
			err = json.Unmarshal([]byte(data), &event)
			if err != nil {
				t.Error(err)
			}
			if expectedEventIdsInOrder[index] != event.EventID {
				t.Error(fmt.Errorf("expected event id to be %s but got %s", expectedEventIdsInOrder[index], event.EventID))
			}
			index++
		}
	}()
	for _, file := range files {
		time.Sleep(file.Timeout * time.Millisecond)
		resp, err := http.Post(fmt.Sprintf("%v/webhooks/payments/orders", url), "application/json", bytes.NewBuffer([]byte(file.Content)))
		if err != nil {
			t.Error(err)
		}
		if resp.Status != "200 OK" {
			t.Error(fmt.Errorf("invalid response status %s", resp.Status))
		}
		err = resp.Body.Close()
		if err != nil {
			t.Error(err)
		}
	}
}

func clearDb(t *testing.T) {
	cfg := config.ReadConfig("../config/local.yaml")
	dbpool, err := pgxpool.New(context.Background(), cfg.Datasource.Url)
	defer dbpool.Close()
	if err != nil {
		t.Fatal("unable to create connection pool", "error", err)
	}
	storage := postgresql.New(dbpool)
	err = storage.DeleteAllFromOrderEvents()
	if err != nil {
		t.Fatal(err)
	}
	err = storage.DeleteAllFromOrders()
	if err != nil {
		t.Fatal(err)
	}
}

func getOrderId(rawEvent string, t *testing.T) string {
	event := model.OrderEvent{}
	err := json.Unmarshal([]byte(rawEvent), &event)
	if err != nil {
		t.Fatal(err)
	}
	return event.OrderID
}

type ScenarioFile struct {
	Index    int
	FileName string
	Content  string
	Timeout  time.Duration
}

func ReadScenarioDir(path string) ([]ScenarioFile, error) {
	dirs, err := scenario.Files.ReadDir(path)
	if err != nil {
		return nil, err
	}
	files := make([]ScenarioFile, 0, len(dirs))
	for _, dir := range dirs {
		name := dir.Name()
		fullPath := filepath.Join(path, name)
		file, err := scenario.Files.ReadFile(fullPath)
		if err != nil {
			return nil, err
		}
		i := strings.Index(name, "_")
		var index int
		if i != -1 {
			index, err = strconv.Atoi(name[:i])
			if err != nil {
				index = 0
			}
		} else {
			index = 0
		}
		lastUnderscore := strings.LastIndex(name, "_")
		lastDot := strings.LastIndex(name, ".")
		var timeout int
		if lastUnderscore != -1 && lastDot != -1 {
			timeout, err = strconv.Atoi(name[lastUnderscore+1 : lastDot])
			if err != nil {
				timeout = 0
			}
		} else {
			timeout = 0
		}
		files = append(files, ScenarioFile{index, name, string(file), time.Duration(timeout)})
	}
	sort.SliceStable(files, func(i, j int) bool {
		return files[i].Index < files[j].Index
	})
	return files, nil
}
