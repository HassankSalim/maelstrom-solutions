package main

import (
	"encoding/json"
	"log"
	"math/rand"
	"os"

	maelstrom "github.com/jepsen-io/maelstrom/demo/go"
)

func getRandomPeerNodes(allNodes []string, n int) []string {
	// Seed the random number generator with the current time
	// rand.Seed(time.Now().UnixNano())

	// Create a copy of the input array to avoid modifying the original
	allNodesCopy := make([]string, len(allNodes))
	copy(allNodesCopy, allNodes)

	// Use a loop to select random elements
	randomElements := make([]string, n)
	for i := 0; i < n; i++ {
		randomIndex := rand.Intn(len(allNodesCopy))
		randomElements[i] = allNodesCopy[randomIndex]

		// Remove the selected element to avoid duplicates
		// (if you want to allow duplicates, you can skip this step)
		allNodesCopy = append(allNodesCopy[:randomIndex], allNodesCopy[randomIndex+1:]...)
	}

	return randomElements
}

func removeNodeFromAllNodes(allNodes []string, nodeToBeRemoved string) []string {
	allNodesCopy := make([]string, len(allNodes))
	copy(allNodesCopy, allNodes)
	// Find the index of the element to remove
	indexToRemove := -1
	for i, element := range allNodesCopy {
		if element == nodeToBeRemoved {
			indexToRemove = i
			break
		}
	}

	// If the element is found, remove it from the array
	if indexToRemove != -1 {
		allNodesCopy = append(allNodesCopy[:indexToRemove], allNodesCopy[indexToRemove+1:]...)
	}

	return allNodesCopy
}

func main() {
	n := maelstrom.NewNode()
	messages := []float64{}
	allNodes := []string{}

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

		body["type"] = "broadcast_ok"
		for _, message := range messages {
			if message == messageVal {
				delete(body, "message")
				return n.Reply(msg, body)
			}
		}

		messages = append(messages, messageVal)
		gossipReqBody := map[string]interface{}{
			"type":    "broadcast",
			"message": messageVal,
		}
		possiblePeerNodes := removeNodeFromAllNodes(allNodes, msg.Src)
		peerNodes := getRandomPeerNodes(possiblePeerNodes, len(possiblePeerNodes))

		logger.Printf("Peer Nodes %v", peerNodes)
		for _, neighbour := range peerNodes {
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
		for _, node := range n.NodeIDs() {
			if node == nodeID {
				continue
			}
			allNodes = append(allNodes, node)
		}
		logger.Printf("All Nodes %v", allNodes)

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
