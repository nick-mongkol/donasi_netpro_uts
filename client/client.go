package main

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"

	"github.com/gorilla/websocket"
)

var ws *websocket.Conn

// Fungsi untuk menghubungkan ke WebSocket
func connectWebSocket() {
	var err error
	ws, _, err = websocket.DefaultDialer.Dial("ws://localhost:8080/ws", nil)
	if err != nil {
		fmt.Println("Error connecting to WebSocket:", err)
		return
	}
	fmt.Println("Connected to WebSocket server")
}

// Fungsi untuk melakukan top-up saldo
func handleTopUp(reader *bufio.Reader) {
	fmt.Print("Masukan username anda beserta jumlah top up (username:jumlah): ")
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	// Connect to TCP server
	conn, err := net.Dial("tcp", "localhost:8081")
	if err != nil {
		fmt.Println("Error connecting to TCP server:", err)
		return
	}
	defer conn.Close()

	// Kirim top-up request
	conn.Write([]byte(input))

	// Menerima respon dari server
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Error reading from TCP connection:", err)
		return
	}
	fmt.Println(string(buffer[:n]))
}

// Fungsi untuk mengecek saldo
func handleCheckBalance(reader *bufio.Reader) {
	fmt.Print("Masukan username untuk mengecek saldo : ")
	username, _ := reader.ReadString('\n')
	username = strings.TrimSpace(username)

	// Connect to UDP server
	conn, err := net.Dial("udp", "localhost:8082")
	if err != nil {
		fmt.Println("Error connecting to UDP server:", err)
		return
	}
	defer conn.Close()

	// Kirim username ke UDP server
	conn.Write([]byte(username))

	// Menerima saldo dari server
	buffer := make([]byte, 1024)
	n, err := conn.Read(buffer)
	if err != nil {
		fmt.Println("Error reading from UDP connection:", err)
		return
	}
	fmt.Println(string(buffer[:n]))
}

// Fungsi untuk melakukan donasi
func handleDonation(reader *bufio.Reader) {
	fmt.Print("Masukan pesan untuk donasi : ")
	message, _ := reader.ReadString('\n')
	message = strings.TrimSpace(message)

	// Kirim pesan donasi ke WebSocket server
	if err := ws.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
		fmt.Println("Error sending message to WebSocket:", err)
	}
}

func main() {
	reader := bufio.NewReader(os.Stdin)

	// Connect to WebSocket server
	connectWebSocket()

	for {
		// Tampilkan menu
		fmt.Println("1. Top-up saldo")
		fmt.Println("2. Cek saldo")
		fmt.Println("3. Donasi")
		fmt.Println("4. Exit")
		fmt.Print("Select option: ")

		option, _ := reader.ReadString('\n')
		option = strings.TrimSpace(option)

		switch option {
		case "1":
			handleTopUp(reader)
		case "2":
			handleCheckBalance(reader)
		case "3":
			handleDonation(reader)
		case "4":
			ws.Close()
			return
		default:
			fmt.Println("Pilih opsi yang benar (1/2/3/4)")
		}
	}
}
