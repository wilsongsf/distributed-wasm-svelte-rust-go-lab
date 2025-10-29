package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"log"
	"math/rand"
	"net/http"
	"sync"
	"time"
	"github.com/google/uuid"
)

var (
	// Em produção, isto deve ficar em KMS e NÃO no código.
	SERVER_SECRET = []byte("server-top-secret-CHANGE_ME")
)

type Job struct {
	Status string `json:"status"`
	Found  bool   `json:"found"`
}

var jobs = struct{
	sync.RWMutex
	m map[string]Job
}{m: map[string]Job{}}

func deriveKey(hint string) []byte {
	mac := hmac.New(sha256.New, SERVER_SECRET)
	mac.Write([]byte(hint))
	full := mac.Sum(nil)
	// truncar para 32 bytes
	return full
}

// XOR encrypt/decrypt -- demo apenas
func xorWithKey(data []byte, key []byte) []byte {
	out := make([]byte, len(data))
	for i := 0; i < len(data); i++ {
		out[i] = data[i] ^ key[i%len(key)]
	}
	return out
}

func makePayload() ([]byte, error) {
	// Payload sensível: intervalos e lista de strings. Aqui geramos demo.
	sensitive := map[string]interface{}{
		"interval": []int{0, 9999},
		"set": []string{"123", "9999", "42"},
	}
	b, err := json.Marshal(sensitive)
	return b, err
}

func jobHandler(w http.ResponseWriter, r *http.Request) {
	// Simples checagem do body { white: true }
	var body map[string]any
	_ = json.NewDecoder(r.Body).Decode(&body)
	if body == nil {
		w.WriteHeader(400)
		w.Write([]byte(`{"error":"bad_request"}`))
		return
	}
	if val, ok := body["white"]; !ok || val != true {
		w.WriteHeader(403)
		w.Write([]byte(`{"error":"forbidden"}`))
		return
	}

	// criar job
	id := uuid.New().String()
	jobs.Lock()
	jobs.m[id] = Job{Status: "queued", Found: false}
	jobs.Unlock()

	// prepara payload cifrado
	keyHint := time.Now().UTC().Format("20060102T150405Z") + "-" + string(rune(rand.Intn(9999)))
	key := deriveKey(keyHint)

	payloadPlain, err := makePayload()
	if err != nil {
		w.WriteHeader(500)
		return
	}
	payloadEnc := xorWithKey(payloadPlain, key)
	payloadB64 := base64.StdEncoding.EncodeToString(payloadEnc)

	// Guarda job e delega processamento lógico — afinal o front fará o trabalho.
	jobs.Lock()
	jobs.m[id] = Job{Status: "dispatched", Found: false}
	jobs.Unlock()

	resp := map[string]string{
		"jobId": id,
		"payload": payloadB64,
		"keyHint": keyHint,
		"wasmUrl": "/wasm/pkg/worker_wasm.js", // em deploy hospede corretamente
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

func resultHandler(w http.ResponseWriter, r *http.Request) {
	var body map[string]string
	_ = json.NewDecoder(r.Body).Decode(&body)
	jobId := body["jobId"]
	resultB64 := body["result"]
	keyHint := body["keyHint"]

	if jobId == "" || resultB64 == "" || keyHint == "" {
		w.WriteHeader(400)
		w.Write([]byte(`{"error":"bad_request"}`))
		return
	}

	// para demo, decifra resultado usando deriveKey
	key := deriveKey(keyHint)
	resBytes, err := base64.StdEncoding.DecodeString(resultB64)
	if err != nil { w.WriteHeader(400); return }
	plain := xorWithKey(resBytes, key)

	// esperamos que plain seja JSON { found: bool }
	var res map[string]any
	_ = json.Unmarshal(plain, &res)
	found := false
	if v, ok := res["found"].(bool); ok { found = v }

	jobs.Lock()
	jobs.m[jobId] = Job{Status: "done", Found: found}
	jobs.Unlock()

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"status":"ok"}`))
}

func statusHandler(w http.ResponseWriter, r *http.Request) {
	id := r.URL.Path[len("/status/"):]
	jobs.RLock()
	j, ok := jobs.m[id]
	jobs.RUnlock()
	if !ok {
		w.WriteHeader(404)
		w.Write([]byte(`{"error":"not_found"}`))
		return
	}
	json.NewEncoder(w).Encode(j)
}

func main() {
	rand.Seed(time.Now().UnixNano())
	http.HandleFunc("/job", jobHandler)
	http.HandleFunc("/result", resultHandler)
	http.HandleFunc("/status/", statusHandler)
	// Servir diretório wasm/ e frontend estático (em deploy usar nginx correto)
	http.Handle("/wasm/", http.StripPrefix("/wasm/", http.FileServer(http.Dir("../wasm/pkg"))))
	http.Handle("/", http.FileServer(http.Dir("../frontend")))

	log.Println("Listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}