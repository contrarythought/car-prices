package dealertree

import (
	"fmt"
	"sync"
)

type Dealer struct {
	Name    string `json:"name"`
	Address string `json:"address"`
	left    *Dealer
	right   *Dealer
}

func NewDealer(name, address string) *Dealer {
	return &Dealer{
		Name:    name,
		Address: address,
		left:    nil,
		right:   nil,
	}
}

type DealerTree struct {
	Root *Dealer
	mu   sync.RWMutex
}

func newDealerTree() *DealerTree {
	return &DealerTree{Root: nil}
}

func createDealerNode(val *Dealer) *Dealer {
	return NewDealer(val.Name, val.Address)
}

func (dt *DealerTree) InsertNode(root, val *Dealer) {
	if root == nil {
		dt.mu.Lock()
		dt.root = createDealerNode(val)
		dt.mu.Unlock()
	} else if val.Name < root.Name {
		dt.insertNode(dt.root.left, val)
	} else {
		dt.insertNode(dt.root.right, val)
	}
}

func (dt *DealerTree) RetrieveDealer(root *Dealer, dealerName string) (*Dealer, error) {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	if root == nil {
		return nil, fmt.Errorf("failed to find %s", dealerName)
	} else if dealerName < root.Name {
		if _, err := dt.RetrieveDealer(root.left, dealerName); err != nil {
			return nil, err
		}
	} else if dealerName > root.Name {
		if _, err := dt.RetrieveDealer(root.right, dealerName); err != nil {
			return nil, err
		}
	}

	return root, nil
}
