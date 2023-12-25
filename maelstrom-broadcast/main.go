package main

import (
	"encoding/json"
	"log"
	"os"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func main() {
	n := maelstrom.NewNode()
	messages := []float64{}
	neigbhourNodes := []string{}

	n.Handle("broadcast", func(msg maelstrom.Message) error {
		logger := log.New(os.Stderr, "", log.Ltime)

		// Unmarshal the message body as an loosely-typed map.
		var body map[string]any

		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}
		messageVal, ok := body["message"].(float64)
		if !ok {
			panic("message is not a number")
		}
		messages = append(messages, messageVal)
		// Update the message type to return back.
		body["type"] = "broadcast_ok"

		gossipReqBody := map[string]interface{}{
			"type":    "broadcast",
			"message": messageVal,
		}
		for _, neighbour := range neigbhourNodes {
			if neighbour == msg.Src {
				continue
			}
			n.RPC(neighbour, gossipReqBody, func(msg maelstrom.Message) error { logger.Printf("Message sent to %s", neighbour); return nil })
		}
		delete(body, "message")

		// Echo the original message back with the updated message type.
		return n.Reply(msg, body)
	})

	n.Handle("read", func(msg maelstrom.Message) error {
		// Unmarshal the message body as an loosely-typed map.
		var body map[string]any
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		// Update the message type to return back.
		body["type"] = "read_ok"
		body["messages"] = messages

		// Echo the original message back with the updated message type.
		return n.Reply(msg, body)
	})

	n.Handle("topology", func(msg maelstrom.Message) error {
		logger := log.New(os.Stderr, "", log.Ltime)
		// Unmarshal the message body as an loosely-typed map.
		var body map[string]any
		if err := json.Unmarshal(msg.Body, &body); err != nil {
			return err
		}

		// Update the message type to return back.
		nodeID := n.ID()
		topology, _ := body["topology"].(map[string]interface{})

		neighbourVals, _ := topology[nodeID].([]interface{})
		for _, neighbourVal := range neighbourVals {
			if val, ok := neighbourVal.(string); ok {
				neigbhourNodes = append(neigbhourNodes, val)
			}
		}
		logger.Printf("Node %s has Neigbhours: %s", nodeID, neigbhourNodes)

		body["type"] = "topology_ok"
		delete(body, "topology")

		// Echo the original message back with the updated message type.
		return n.Reply(msg, body)
	})

	if err := n.Run(); err != nil {
		log.Printf("ERROR: %s", err)
		os.Exit(1)
	}
}
