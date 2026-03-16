package service

import (
	"eth-sweeper/model"
	"fmt"
	"strconv"
	"strings"
)

type GraphService struct {
	etherscan *EtherscanClient
}

func NewGraphService(etherscan *EtherscanClient) *GraphService {
	return &GraphService{etherscan: etherscan}
}

func (s *GraphService) BuildGraph(centerAddress string) (*model.GraphResponse, error) {
	centerAddress = strings.ToLower(centerAddress)

	txs, err := s.etherscan.GetAllTransactionsForGraph(centerAddress)
	if err != nil {
		return nil, fmt.Errorf("failed to get transactions for %s: %w", centerAddress, err)
	}

	nodeMap := map[string]*model.GraphNode{}
	edgeMap := map[string]*model.GraphEdge{}

	nodeMap[centerAddress] = &model.GraphNode{
		ID:       centerAddress,
		Label:    shortenAddress(centerAddress),
		IsCenter: true,
		TxCount:  len(txs),
	}

	for _, tx := range txs {
		from := strings.ToLower(tx.From)
		to := strings.ToLower(tx.To)
		if from == "" || to == "" {
			continue
		}

		counterparty := to
		if from != centerAddress {
			counterparty = from
		}

		if _, exists := nodeMap[counterparty]; !exists {
			node := &model.GraphNode{
				ID:       counterparty,
				Label:    shortenAddress(counterparty),
				IsCenter: false,
				TxCount:  0,
			}
			if label := LookupAddress(counterparty); label != nil {
				node.Tag = label.Tag
				node.TagName = label.Name
				node.Label = label.Name
			}
			nodeMap[counterparty] = node
		}
		nodeMap[counterparty].TxCount++

		edgeKey := from + "->" + to
		if edge, exists := edgeMap[edgeKey]; exists {
			edge.TxCount++
			edge.Value = addValues(edge.Value, tx.Value)
		} else {
			edgeMap[edgeKey] = &model.GraphEdge{
				Source:  from,
				Target:  to,
				Value:   tx.Value,
				TxCount: 1,
			}
		}
	}

	nodes := make([]model.GraphNode, 0, len(nodeMap))
	for _, n := range nodeMap {
		if !n.IsCenter {
			n.IsContract = s.etherscan.IsContract(n.ID)
		}
		nodes = append(nodes, *n)
	}

	edges := make([]model.GraphEdge, 0, len(edgeMap))
	for _, e := range edgeMap {
		edges = append(edges, *e)
	}

	return &model.GraphResponse{
		Nodes: nodes,
		Edges: edges,
	}, nil
}

func shortenAddress(addr string) string {
	if len(addr) <= 10 {
		return addr
	}
	return addr[:6] + "..." + addr[len(addr)-4:]
}

func addValues(a, b string) string {
	fa := parseFloatSafe(a)
	fb := parseFloatSafe(b)
	sum := fa + fb
	if sum == 0 {
		return "0"
	}
	return strconv.FormatFloat(sum, 'f', -1, 64)
}

func parseFloatSafe(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}
