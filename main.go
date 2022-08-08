package main

import (
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/gorilla/mux"
)

type Block struct {
	Pos       int
	Data      MovieCheckout
	Timestamp string
	Hash      string
	PrevHash  string
}

type MovieCheckout struct {
	MovieID      string `json: "movie_id"`
	User         string `json: "user"`
	CheckoutDate string `json: "checkout_date`
	IsGenesis    bool   `json: "is_genesis"`
}

type Movie struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Director    string `json:"director"`
	ReleaseDate string `json:"release_date"`
}

type Blockchain struct {
	blocks []*Block
}

var BlockChain *Blockchain

func (b *Block) generateHash() {
	bytes, _ := json.Marshal(b.Data)

	data := string(b.Pos) + b.Timestamp + string(bytes) + b.PrevHash

	hash := sha256.New()
	hash.Write([]byte(data))
	b.Hash = hex.EncodeToString(hash.Sum(nil))
}
func CreateBlock(prevBlock *Block, checkoutItem MovieCheckout) *Block {
	block := &Block{}
	block.Pos = prevBlock.Pos + 1
	block.Timestamp = time.Now().String()
	block.PrevHash = prevBlock.Hash
	block.generateHash()

	return block
}

func (b *Block) validateHash(hash string) bool {
	b.generateHash()
	if b.Hash != hash {
		return false
	}

	return true
}

func (bc *Blockchain) AddBlock(data MovieCheckout) {

	prevBlock := bc.blocks[len(bc.blocks)-1]
	block := CreateBlock(prevBlock, data)

	if validBlock(block, prevBlock) {
		bc.blocks = append(bc.blocks, block)
	}

}

func validBlock(block, prevBlock *Block) bool {
	if prevBlock.Hash != block.PrevHash {
		return false
	}

	if !block.validateHash(block.Hash) {
		return false
	}

	if prevBlock.Pos+1 != block.Pos {
		return false
	}

	return true
}

func writeBlock(w http.ResponseWriter, r *http.Request) {
	var checkoutItem MovieCheckout

	if err := json.NewDecoder(r.Body).Decode(&checkoutItem); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("could not write block:%v", err)
		w.Write([]byte("could not write block"))
	}

	BlockChain.AddBlock(checkoutItem)
}

func newMovie(w http.ResponseWriter, r *http.Request) {
	var movie Movie

	if err := json.NewDecoder(r.Body).Decode(&movie); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("could not create:%v", err)
		w.Write([]byte("could not reate book"))
		return
	}

	h := md5.New()
	io.WriteString(h, movie.ID+movie.ReleaseDate)
	movie.ID = fmt.Sprintf("%x", h.Sum(nil))

	resp, err := json.MarshalIndent(movie, "", " ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		log.Printf("could not marshal payload: %v", err)
		w.Write([]byte("could not save book data"))
		return
	}
	w.WriteHeader(http.StatusOK)
	w.Write(resp)
}

func GenesisBlock() *Block {
	return CreateBlock(&Block{}, MovieCheckout{IsGenesis: true})
}

func NewBlockchain() *Blockchain {
	return &Blockchain{[]*Block{GenesisBlock()}}
}

func getBlockchain(w http.ResponseWriter, r *http.Request) {
	jbytes, err := json.MarshalIndent(BlockChain.blocks, "", " ")
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(err)
		return
	}
	io.WriteString(w, string(jbytes))
}

func main() {

	BlockChain = NewBlockchain()

	r := mux.NewRouter()
	r.HandleFunc("/", getBlockchain).Methods("GET")
	r.HandleFunc("/", writeBlock).Methods("POST")
	r.HandleFunc("/new", newMovie).Methods("POST")

	go func() {
		for _, block := range BlockChain.blocks {
			fmt.Printf("Prev. hash: %x\n", block.PrevHash)
			bytes, _ := json.MarshalIndent(block.Data, "", " ")
			fmt.Printf("Data:%v\n", string(bytes))
			fmt.Println()
		}
	}()

	log.Println("Listening on PORT: 3000")

	log.Fatal(http.ListenAndServe(":3000", r))
}
