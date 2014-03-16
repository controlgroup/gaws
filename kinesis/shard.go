package kinesis

import (
	"bytes"
	"encoding/json"
	"net/http"

	"github.com/controlgroup/gaws"
)

// HashKeyRange is the range of hash keys for a shard. It is used in the Shard type.
type HashKeyRange struct {
	EndingHashKey   string
	StartingHashKey string
}

// SequenceNumberRange is the range of sequence numbers used in a shard. It is used in the Shard type.
type SequenceNumberRange struct {
	EndingSequenceNumber   string
	StartingSequenceNumber string
}

// Shard is a shard in a Kinesis stream.
type Shard struct {
	AdjacentParentShardId string
	HashKeyRange          HashKeyRange
	ParentShardId         string
	SequenceNumberRange   SequenceNumberRange
	ShardId               string
	stream                *Stream
}

type getShardIteratorResponse struct {
	ShardIterator string
}

type getShardIteratorRequest struct {
	ShardId                string
	ShardIteratorType      string
	StartingSequenceNumber string `json:",omitempty"`
	StreamName             string
}

// GetShardIterator gets a shard iterator from the shard. It takes a type, which is one of: AT_SEQUENCE_NUMBER, AFTER_SEQUENCE_NUMBER, TRIM_HORIZON, or LATEST and an optional sequence number to start on.
// See http://docs.aws.amazon.com/kinesis/latest/APIReference/API_GetShardIterator.html for more details.
func (s *Shard) GetShardIterator(shardIteratorType string, startingSequenceNumber string) (string, error) {
	url := s.stream.Service.Endpoint
	result := getShardIteratorResponse{}

	body := getShardIteratorRequest{ShardId: s.ShardId, ShardIteratorType: shardIteratorType, StartingSequenceNumber: startingSequenceNumber, StreamName: s.stream.Name}

	bodyAsJson, err := json.Marshal(body)
	payload := bytes.NewReader(bodyAsJson)

	req, err := http.NewRequest("POST", url, payload)

	if err != nil {
		return "", err
	}

	req.Header.Set("X-Amz-Target", "Kinesis_20131202.GetShardIterator")
	req.Header.Set("Content-Type", "application/x-amz-json-1.1")

	resp, err := gaws.SendAWSRequest(req)
	if err != nil {
		return "", err
	}

	err = json.Unmarshal(resp, &result)
	if err != nil {
		return "", err
	}
	return result.ShardIterator, err
}
