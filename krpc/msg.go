package krpc

// Msg represents messages that nodes in the network send to each other as specified by the protocol.
// They are also referred to as the KRPC messages.
// There are three types of messages: QUERY, RESPONSE, ERROR
// The message is a dictionary that is then
// "bencoded" (serialization & compression format adopted by the BitTorrent)
// and sent via the UDP connection to peers.
//
// A KRPC message is a single dictionary with two keys common to every message and additional keys depending on the type of message.
// Every message has a key "t" with a string value representing a transaction ID.
// This transaction ID is generated by the querying node and is echoed in the response, so responses
// may be correlated with multiple queries to the same node. The transaction ID should be encoded as a short string of binary numbers, typically 2 characters are enough as they cover 2^16 outstanding queries. The other key contained in every KRPC message is "y" with a single character value describing the type of message. The value of the "y" key is one of "q" for query, "r" for response, or "e" for error.
// 3 message types:  QUERY, RESPONSE, ERROR
type Msg struct {
	Q        string   `bencode:"q,omitempty"` // Query method (one of 4: "ping", "find_node", "get_peers", "announce_peer")
	A        *MsgArgs `bencode:"a,omitempty"` // named arguments sent with a query
	T        string   `bencode:"t"`           // required: transaction ID
	Y        string   `bencode:"y"`           // required: type of the message: q for QUERY, r for RESPONSE, e for ERROR
	R        *Return  `bencode:"r,omitempty"` // RESPONSE type only
	E        *Error   `bencode:"e,omitempty"` // ERROR type only
	IP       NodeAddr `bencode:"ip,omitempty"`
	ReadOnly bool     `bencode:"ro,omitempty"` // BEP 43. Sender does not respond to queries.
}

type MsgArgs struct {
	ID       ID `bencode:"id"`                  // ID of the querying Node
	InfoHash ID `bencode:"info_hash,omitempty"` // InfoHash of the torrent
	Target   ID `bencode:"target,omitempty"`    // ID of the node sought
	// Token received from an earlier get_peers query. Also used in a BEP 44 put.
	Token       string `bencode:"token,omitempty"`
	Port        *int   `bencode:"port,omitempty"`         // Sender's torrent port
	ImpliedPort bool   `bencode:"implied_port,omitempty"` // Use senders apparent DHT port
	Want        []Want `bencode:"want,omitempty"`         // Contains strings like "n4" and "n6" from BEP 32.
	NoSeed      int    `bencode:"noseed,omitempty"`       // BEP 33
	Scrape      int    `bencode:"scrape,omitempty"`       // BEP 33

	// BEP 44
	V interface{} `bencode:"v,omitempty"`
}

type Want string

const (
	WantNodes  Want = "n4"
	WantNodes6 Want = "n6"
)

// BEP 51 (DHT Infohash Indexing)
type Bep51Return struct {
	Interval *int64 `bencode:"interval,omitempty"`
	Num      *int64 `bencode:"num,omitempty"`
	// Nodes supporting this extension should always include the samples field in the response, even
	// when it is zero-length. This lets indexing nodes to distinguish nodes supporting this
	// extension from those that respond to unknown query types which contain a target field [2].
	Samples *CompactInfohashes `bencode:"samples,omitempty"`
}

type Return struct {
	// All returns are supposed to contain an ID, but what if they don't?
	ID ID `bencode:"id"` // ID of the queried (and responding) node

	// K closest nodes to the requested target. Included in responses to queries that imply
	// traversal, for example get_peers, find_nodes, get, sample_infohashes.
	Nodes  CompactIPv4NodeInfo `bencode:"nodes,omitempty"`
	Nodes6 CompactIPv6NodeInfo `bencode:"nodes6,omitempty"`

	Token  *string    `bencode:"token,omitempty"`  // Token for future announce_peer or put (BEP 44)
	Values []NodeAddr `bencode:"values,omitempty"` // Torrent peers

	// BEP 33 (scrapes)
	BFsd *ScrapeBloomFilter `bencode:"BFsd,omitempty"`
	BFpe *ScrapeBloomFilter `bencode:"BFpe,omitempty"`

	Bep51Return

	// BEP 44 get
	V interface{} `bencode:"v,omitempty"`
}

func (r Return) ForAllNodes(f func(NodeInfo)) {
	for _, n := range r.Nodes {
		f(n)
	}
	for _, n := range r.Nodes6 {
		f(n)
	}
}

// The node ID of the source of this Msg. Returns nil if it isn't present.
// TODO: Can we verify Msgs more aggressively so this is guaranteed to return
// a valid ID for a checked Msg?
func (m Msg) SenderID() *ID {
	switch m.Y {
	case "q":
		if m.A == nil {
			return nil
		}
		return &m.A.ID
	case "r":
		if m.R == nil {
			return nil
		}
		return &m.R.ID
	}
	return nil
}

func (m Msg) Error() *Error {
	if m.Y != "e" {
		return nil
	}
	return m.E
}
