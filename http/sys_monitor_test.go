package http

import (
	"bufio"
	"fmt"
	"github.com/hashicorp/go-cleanhttp"
	"github.com/hashicorp/vault/vault"
	"io/ioutil"
	"strings"
	"testing"
	"time"
)

func TestSysMonitor_get(t *testing.T) {
	t.Parallel()
	core := vault.TestCore(t)
	ln, addr := TestServer(t, core)
	defer ln.Close()

	t.Run("unknown log level", func(t *testing.T) {
		client := cleanhttp.DefaultClient()
		resp, err := client.Get(addr + "/v1/sys/monitor?log_level=haha")
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		code := resp.StatusCode
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			t.Fatalf("err: %s", err)
		}

		if code != 400 {
			t.Fatalf("expected to receive a 400, got %v instead\n", code)
		}

		substring := "unknown log level"
		if !strings.Contains(string(body), substring) {
			t.Fatalf("expected a message containing: %s, got %s instead\n", substring, body)
		}
	})

	t.Run("stream unstructured logs", func(t *testing.T) {
	//	c, err := api.NewClient(&api.Config{
	//		HttpClient: http.DefaultClient,
	//	})
	//
	//	if err != nil {
	//		t.Fatal(err)
	//	}
	//
	//	var logCh chan string
	//	stopCh := make(chan struct{})
	//	defer close(stopCh)
	//	logCh, err = c.Sys().Monitor("DEBUG", stopCh)
	//
	//	if err != nil {
	//		t.Fatal(err)
	//	}
	//
	//	// Make a request to generate some logs
	//	_, err = http.Get(addr + "/v1/sys/health")
	//	t.Log("finished making the health request")
	//	if err != nil {
	//		t.Fatalf("err: %s", err)
	//	}
	//
	//OUTER:
	//	for {
	//		select {
	//		case log := <-logCh:
	//			t.Logf("log = %s\n", log)
	//			if strings.Contains(log, "[DEBUG]") {
	//				break OUTER
	//			}
	//		case <-time.After(2 * time.Second):
	//			t.Fatal("failed to get a DEBUG message")
	//		}
	//	}
	//
	//	t.Fatal("aw man")


		// ============================================

		stopCh := make(chan struct{})

		// Make requests that generate logs
		go func() {
			client := cleanhttp.DefaultClient()

			for {
				select {
				case <-stopCh:
					t.Log("received input on stopCh, returning")
					return
				default:
				}

				fmt.Println("bye")

				_, err := client.Get(addr + "/v1/sys/health")
				t.Log("finished making the health request")

				if err != nil {
					t.Fatalf("err: %s", err)
				}

				time.Sleep(1 * time.Second)
			}
		}()

		// Watch for logs that match what we want
		go func() {
			client := cleanhttp.DefaultClient()

			for {
				select {
				case <-stopCh:
					t.Log("received input on stopCh, returning")
					return
				default:
				}

				fmt.Println("hi")

				resp, err := client.Get(addr + "/v1/sys/monitor?log_level=debug")

				if err != nil {
					t.Fatalf("err: %s", err)
				}

				scanner := bufio.NewScanner(resp.Body)

				if scanner.Scan() {
					if text := scanner.Text(); text != "" {
						t.Logf("body = %s", text)

						if strings.Contains(text, "[DEBUG]") {
							t.Log("found DEBUG logs")
							stopCh <- struct{}{}
							return
						}
					} else {
						t.Fatal("no text?")
					}
				} else {
					t.Fatal("scanner won't scan")
				}

				_ = resp.Body.Close()
				time.Sleep(1 * time.Second)
			}
		}()


		select {
		case <-time.After(5 * time.Second):
			t.Log("it's been 5 seconds")
			stopCh <-struct{}{}
			t.Fatal("Failed to get a DEBUG message after 5 seconds")
		}

		//go func() {
		//	resp, err := http.Get(addr + "/v1/sys/monitor?log_level=debug")
		//	defer resp.Body.Close()
		//
		//	if err != nil {
		//		t.Fatalf("err: %s", err)
		//	}
		//
		//OUTER:
		//	for {
		//		select {
		//		case <-time.After(2 * time.Second):
		//			t.Fatalf("failed to get a DEBUG message after 2 seconds")
		//		default:
		//			t.Log("nothing to do here")
		//		}
		//
		//		scanner := bufio.NewScanner(resp.Body)
		//		t.Log("lol yes")
		//
		//		if scanner.Scan() {
		//			t.Log("scan worked")
		//			if text := scanner.Text(); text != "" {
		//				t.Logf("text = %s\n", text)
		//
		//				if strings.Contains(text, "[DEBUG]") {
		//					t.Log("HOORAY")
		//					break OUTER
		//				}
		//			} else {
		//				t.Logf("text = %s\n", text)
		//			}
		//		} else {
		//			t.Log("scanner can't scan any more")
		//		}
		//	}
		//}()
		//
		//t.Log("good times")
		//
		//// Make a request to generate some logs
		//_, err := http.Get(addr + "/v1/sys/health")
		//t.Log("finished making the health request")
		//if err != nil {
		//	t.Fatalf("err: %s", err)
		//}
		//time.Sleep(5 * time.Second)
		//t.Fatal("finished sleeping")
	})

	//t.Run("stream JSON logs", func(t *testing.T) {
	//
	//})

	//t.Run("stream unstructured logs", func(t *testing.T) {
	//	// Try to stream logs until we see the expected log line
	//	retry.Run(t, func(r *retry.R) {
	//		req, _ := http.NewRequest("GET", "/v1/agent/monitor?loglevel=debug", nil)
	//		cancelCtx, cancelFunc := context.WithCancel(context.Background())
	//		req = req.WithContext(cancelCtx)
	//
	//		resp := httptest.NewRecorder()
	//		errCh := make(chan error)
	//		go func() {
	//			_, err := a.srv.AgentMonitor(resp, req)
	//			errCh <- err
	//		}()
	//
	//		args := &structs.ServiceDefinition{
	//			Name: "monitor",
	//			Port: 8000,
	//			Check: structs.CheckType{
	//				TTL: 15 * time.Second,
	//			},
	//		}
	//
	//		registerReq, _ := http.NewRequest("PUT", "/v1/agent/service/register", jsonReader(args))
	//		if _, err := a.srv.AgentRegisterService(nil, registerReq); err != nil {
	//			t.Fatalf("err: %v", err)
	//		}
	//
	//		// Wait until we have received some type of logging output
	//		require.Eventually(t, func() bool {
	//			return len(resp.Body.Bytes()) > 0
	//		}, 3*time.Second, 100*time.Millisecond)
	//
	//		cancelFunc()
	//		err := <-errCh
	//		require.NoError(t, err)
	//
	//		got := resp.Body.String()
	//
	//		// Only check a substring that we are highly confident in finding
	//		want := "Synced service: service="
	//		if !strings.Contains(got, want) {
	//			r.Fatalf("got %q and did not find %q", got, want)
	//		}
	//	})
	//})
	//
	//t.Run("stream JSON logs", func(t *testing.T) {
	//	// Try to stream logs until we see the expected log line
	//	retry.Run(t, func(r *retry.R) {
	//		req, _ := http.NewRequest("GET", "/v1/agent/monitor?loglevel=debug&logjson", nil)
	//		cancelCtx, cancelFunc := context.WithCancel(context.Background())
	//		req = req.WithContext(cancelCtx)
	//
	//		resp := httptest.NewRecorder()
	//		errCh := make(chan error)
	//		go func() {
	//			_, err := a.srv.AgentMonitor(resp, req)
	//			errCh <- err
	//		}()
	//
	//		args := &structs.ServiceDefinition{
	//			Name: "monitor",
	//			Port: 8000,
	//			Check: structs.CheckType{
	//				TTL: 15 * time.Second,
	//			},
	//		}
	//
	//		registerReq, _ := http.NewRequest("PUT", "/v1/agent/service/register", jsonReader(args))
	//		if _, err := a.srv.AgentRegisterService(nil, registerReq); err != nil {
	//			t.Fatalf("err: %v", err)
	//		}
	//
	//		// Wait until we have received some type of logging output
	//		require.Eventually(t, func() bool {
	//			return len(resp.Body.Bytes()) > 0
	//		}, 3*time.Second, 100*time.Millisecond)
	//
	//		cancelFunc()
	//		err := <-errCh
	//		require.NoError(t, err)
	//
	//		// Each line is output as a separate JSON object, we grab the first and
	//		// make sure it can be unmarshalled.
	//		firstLine := bytes.Split(resp.Body.Bytes(), []byte("\n"))[0]
	//		var output map[string]interface{}
	//		if err := json.Unmarshal(firstLine, &output); err != nil {
	//			t.Fatalf("err: %v", err)
	//		}
	//	})
	//})
}
