package main

import (
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/websocket"
)

var (
	clients   = make(map[string]float64)       // Map untuk menyimpan saldo per username
	mu        sync.Mutex                       // Mutex untuk thread-safety
	wsClients = make(map[*websocket.Conn]bool) // Simpan WebSocket clients
	upgrader  = websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool {
			return true // Allow all origins
		},
	}
)

type Donation struct {
	Name    string  `json:"name"`
	Amount  float64 `json:"amount"`
	Message string  `json:"message"`
}

func main() {
	// Mulai server TCP untuk top-up
	go startTCPServer()

	// Mulai server UDP untuk cek saldo
	go startUDPServer()

	// Mulai WebSocket server untuk donasi
	http.HandleFunc("/ws", handleWebSocket)
	go func() {
		if err := http.ListenAndServe(":8080", nil); err != nil {
			fmt.Println("Error starting WebSocket server:", err)
		}
	}()

	// Tunggu agar server tetap berjalan
	select {}
}

// Fungsi untuk memulai server TCP (top-up)
func startTCPServer() {
	// TCP server untuk top-up saldo
	listener, err := net.Listen("tcp", ":8081")
	if err != nil {
		fmt.Println("Error starting TCP server:", err)
		return
	}
	defer listener.Close()

	fmt.Println("TCP server for top-up started at :8081")

	for {
		conn, err := listener.Accept()
		if err != nil {
			fmt.Println("Error accepting TCP connection:", err)
			continue
		}

		// Tangani permintaan top-up
		go handleTopUp(conn)
	}
}

// Fungsi untuk menangani top-up saldo
func handleTopUp(conn net.Conn) {
	defer conn.Close()

	// Terima data top-up
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Error reading from connection:", err)
		return
	}

	// Proses data top-up
	data := strings.Split(string(buffer[:n]), ":")
	if len(data) != 2 {
		conn.Write([]byte("Invalid data format. Format: <username>:<amount>\n"))
		return
	}

	username := data[0]
	amount := data[1]

	// Konversi amount ke float64
	var amountFloat float64
	_, err = fmt.Sscanf(amount, "%f", &amountFloat)
	if err != nil {
		conn.Write([]byte("Invalid amount format. Amount should be a valid number.\n"))
		return
	}

	// Lock untuk thread-safety
	mu.Lock()
	defer mu.Unlock()

	// Cek apakah username sudah ada di map
	_, exists := clients[username]
	if !exists {
		clients[username] = 0 // Inisialisasi saldo
	}

	// Update saldo
	clients[username] += amountFloat

	// Kirimkan respons ke TCP client
	conn.Write([]byte(fmt.Sprintf("Top-up berhasil. Jumlah Saldo %s: %.2f\n", username, clients[username])))

	// Broadcast perubahan saldo ke semua WebSocket client
	broadcast(username, amountFloat, fmt.Sprintf("Top-up berhasil. Jumlah Saldo : %.2f", clients[username]))
}

// Fungsi untuk memulai server UDP (cek saldo)
func startUDPServer() {
	// UDP server untuk cek saldo
	addr, err := net.ResolveUDPAddr("udp", ":8082")
	if err != nil {
		fmt.Println("Error resolving UDP address:", err)
		return
	}

	conn, err := net.ListenUDP("udp", addr)
	if err != nil {
		fmt.Println("Error starting UDP server:", err)
		return
	}
	defer conn.Close()

	fmt.Println("UDP server for balance check started at :8082")

	for {
		buffer := make([]byte, 1024)
		n, addr, err := conn.ReadFromUDP(buffer)
		if err != nil {
			fmt.Println("Error reading from UDP:", err)
			continue
		}

		// Proses data cek saldo
		username := string(buffer[:n])

		// Cek saldo
		mu.Lock()
		balance, exists := clients[username]
		mu.Unlock()

		var response string
		if exists {
			response = fmt.Sprintf("Saldo milik %s: %.2f", username, balance)
		} else {
			response = fmt.Sprintf("Username %s not found.", username)
		}

		// Kirimkan saldo ke client
		conn.WriteToUDP([]byte(response), addr)
	}
}

// Fungsi untuk menangani WebSocket client
func handleWebSocket(w http.ResponseWriter, r *http.Request) {
	// Upgrade HTTP connection to WebSocket connection
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		fmt.Println("Error upgrading to WebSocket:", err)
		return
	}
	defer conn.Close()

	// Tambahkan client WebSocket ke map
	wsClients[conn] = true
	fmt.Println("New WebSocket client connected")

	// Mendengarkan pesan dari WebSocket client
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			fmt.Println("Error reading WebSocket message:", err)
			delete(wsClients, conn)
			break
		}

		// Parse pesan JSON jika pesan berisi donasi
		var donation Donation
		if err := json.Unmarshal(message, &donation); err == nil {
			// Jika valid, broadcast donasi ke semua WebSocket client
			broadcast(donation.Name, donation.Amount, donation.Message) // Memanggil broadcast dengan argumen yang benar
			fmt.Printf("Broadcasting donation: Donasi dari %s: Rp%.2f. Pesan: %s\n", donation.Name, donation.Amount, donation.Message)
		} else {
			fmt.Println("Error parsing donation:", err)
		}
	}
}

// Fungsi untuk mengirim pesan ke semua WebSocket client
func broadcast(name string, amount float64, message string) {
	for client := range wsClients {
		donation := map[string]interface{}{
			"name":    name,
			"amount":  amount,
			"message": message,
		}

		donationJSON, err := json.Marshal(donation)
		if err != nil {
			fmt.Println("Error marshalling donation:", err)
			continue
		}

		err = client.WriteMessage(websocket.TextMessage, donationJSON)
		if err != nil {
			fmt.Println("Error sending message to WebSocket client:", err)
			client.Close()
			delete(wsClients, client)
		}
	}
	fmt.Printf("Total WebSocket clients: %d\n", len(wsClients))
}
