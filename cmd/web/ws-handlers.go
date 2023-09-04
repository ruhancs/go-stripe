package main

import (
	"fmt"
	"net/http"

	"github.com/gorilla/websocket"
)

//pegar conexao com websocket
type WebSocketConnection struct {
	*websocket.Conn
}

type WsPayload struct {
	Action string `json:"action"`
	Message string `json:"message"`
	UserName string `json:"username"`
	MessageType string `json:"message_type"`
	UserID int `json:"user_id"`
	Conn WebSocketConnection `json:"-"`
}

type WsJsonResponse struct {
	Action string `json:"action"`
	Message string `json:"message"`
	UserID int `json:"user_id"`
}

var upgradeConnection = websocket.Upgrader{
	ReadBufferSize: 1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {return true},
}

var clients = make(map[WebSocketConnection]string)

var wsChan = make(chan WsPayload)

func (app *application) WsEndpoint(w http.ResponseWriter, r *http.Request) {
	ws, err := upgradeConnection.Upgrade(w, r, nil)
	if err != nil {
		app.errorLog.Println(err)
		return
	}
	
	app.infolog.Println(fmt.Sprintf("Client connected from %s", r.RemoteAddr))
	
	var response WsJsonResponse
	response.Message = "Connected to server"
	
	err = ws.WriteJSON(response)
	if err != nil {
		app.errorLog.Println(err)
		return
	}

	//conexao do ws
	conn := WebSocketConnection{Conn: ws}
	//map das conexoes
	clients[conn]= ""

	go app.ListenForWs(&conn)
}

func (app *application) ListenForWs(conn *WebSocketConnection) {
	//se algo der errado a aplicacao nao morre
	defer func() {
		if r := recover(); r != nil {
			app.errorLog.Println("ERROR", fmt.Sprintf("%v",r))
		}
	}()

	var payload WsPayload

	//executar para sempre
	for {
		err := conn.ReadJSON(&payload)
		if err != nil {
			//nao faz nada em caso de erro
		} else {
			//insere a conexao no payload e envia para o canal
			payload.Conn = *conn
			wsChan <- payload
		}
	}
}

//escuta o canal do ws
func (app *application) ListenToWsChannel() {
	var response WsJsonResponse
	for {
		event := <-wsChan
		switch event.Action {
			//caso o evento no canal de ws for deleteUser
		case "deleteUser":
			response.Action = "logout"//quando o usuario é delete faz o logout do usuario
			response.Message = "Your account has been deleted"
			response.UserID = event.UserID //enviar o id do usuario no evento
			app.broadcastToAll(response)//informa para todos conectado no canal que usuario foi deletado

		default:
		}
	}
}

func (app *application) broadcastToAll(response WsJsonResponse) {
	//rodar por todo o map de clients conectados no ws para enviar a msg
	for client := range clients {
		err := client.WriteJSON(response)
		if err != nil {
			app.errorLog.Printf("websocket err on %s: %s", response.Action, err)
			_ = client.Close()
			delete(clients, client)//tirar o client do map de conexoes
		}
	}
}