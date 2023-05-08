package brokers

import (
	"context"
	"encoding/json"
	"fmt"
	"mock-server/internal/coderun"
	"mock-server/internal/database"

	zlog "github.com/rs/zerolog/log"
)

const EMPTY_MAPPER = ""

type ESBRecordError struct {
	err string
}

func NewESBRecordError(err string) *ESBRecordError {
	return &ESBRecordError{err}
}

func (err ESBRecordError) Error() string {
	return err.err
}

func runMapper(mapper_name string, msgs []string) ([]string, error) {
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

func submitToESB(pool_name_in string, msgs []string) error {
	record, err := database.GetESBRecord(context.TODO(), pool_name_in)
	if err == database.ErrNoSuchRecord {
		zlog.Warn().Str("pool_in", pool_name_in).Msg("no registered esb records, skipping")
		return nil
	} else if err != nil {
		return err
	}

	zlog.Info().Str("pool_in", pool_name_in).Str("pool_out", record.PoolNameOut).Msg("found esb record")

	handler, err := GetMessagePool(record.PoolNameOut)
	if err != nil {
		return err
	}

	if record.MapperScriptName != EMPTY_MAPPER {
		msgs, err = runMapper(record.MapperScriptName, msgs)
		if err != nil {
			return err
		}
	}

	handler.NewWriteTask(msgs).Schedule()

	return nil
}

func addEsbRecord(record database.ESBRecord) error {
	err := database.AddESBRecord(context.TODO(), record)
	if err == database.ErrDuplicateKey {
		return NewESBRecordError(fmt.Sprintf("esb record: %s already exists", record.PoolNameIn))
	}

	return err
}

func AddEsbRecord(pool_name_in string, pool_name_out string) error {
	return addEsbRecord(database.ESBRecord{
		PoolNameIn:       pool_name_in,
		PoolNameOut:      pool_name_out,
		MapperScriptName: EMPTY_MAPPER,
	})
}

func AddEsbRecordWithMapper(pool_name_in string, pool_name_out string, mapperScriptName string) error {
	return addEsbRecord(database.ESBRecord{
		PoolNameIn:       pool_name_in,
		PoolNameOut:      pool_name_out,
		MapperScriptName: mapperScriptName,
	})
}

func RemoveEsbRecord(pool_name_in string) error {
	err := database.RemoveESBRecord(context.TODO(), pool_name_in)
	if err == database.ErrNoSuchRecord {
		return NewESBRecordError(fmt.Sprintf("esb record: %s is not registered", pool_name_in))
	}
	return err
}
