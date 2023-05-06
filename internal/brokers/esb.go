package brokers

import (
	"encoding/json"
	"fmt"
	"mock-server/internal/coderun"
	"mock-server/internal/util"

	zlog "github.com/rs/zerolog/log"
)

var Esb = &esb{}

type esbRecord struct {
	pool_name_out string
	mapper        string
	use_mapper    bool
}

type esb struct {
	records util.SyncMap[string, esbRecord]
}

func (e *esb) Init() {
	e.records = util.NewSyncMap[string, esbRecord]()
	// fetch db
}

func (e *esb) runMapper(mapper_name string, msgs []string) ([]string, error) {
	worker, err := coderun.WorkerWatcher.BorrowWorker()
	if err != nil {
		return nil, err
	}

	defer worker.Return()

	out, err := worker.RunScript("mapper", mapper_name, coderun.NewMapperArgs(msgs))
	if err != nil {
		return nil, err
	}

	zlog.Debug().Str("mapped_msgs", string(out)).Msg("got mapped data")

	var mappedMsgs []string
	if err = json.Unmarshal(out, &mappedMsgs); err != nil {
		return nil, err
	}

	return mappedMsgs, nil
}

func (e *esb) submit(pool_name_in string, msgs []string) error {
	record, exists := e.records.Get(pool_name_in)
	if !exists {
		zlog.Warn().Str("pool_in", pool_name_in).Msg("no registered esb records, skipping")
		return nil
	}

	zlog.Info().Str("pool_in", pool_name_in).Str("pool_out", record.pool_name_out).Msg("found esb record")

	handler, err := MPRegistry.GetMessagePool(record.pool_name_out)
	if err != nil {
		return err
	}

	if record.use_mapper {
		msgs, err = e.runMapper(record.mapper, msgs)
		if err != nil {
			return err
		}
	}

	handler.NewWriteTask(msgs).Schedule()

	return nil
}

func (e *esb) addEsbRecord(pool_name_in string, record esbRecord) error {
	if e.records.Contains(pool_name_in) {
		return fmt.Errorf("esb record: %s already exists", pool_name_in)
	}

	e.records.Add(pool_name_in, record)
	// save to db

	return nil
}

func (e *esb) AddEsbRecord(pool_name_in string, pool_name_out string) error {
	return e.addEsbRecord(pool_name_in, esbRecord{
		pool_name_out: pool_name_out,
		mapper:        "",
		use_mapper:    false,
	})
}

func (e *esb) AddEsbRecordWithMapper(pool_name_in string, pool_name_out string, mapper string) error {
	return e.addEsbRecord(pool_name_in, esbRecord{
		pool_name_out: pool_name_out,
		mapper:        mapper,
		use_mapper:    true,
	})
}

func (e *esb) RemoveEsbRecord(pool_name_in string) error {
	if !e.records.Contains(pool_name_in) {
		return fmt.Errorf("esb record: %s is not registered", pool_name_in)
	}

	e.records.Remove(pool_name_in)
	// remove from db

	return nil
}
