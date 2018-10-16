
package blocks

import (
  "time"
  "encoding/json"
  "fmt"
  "github.com/go-errors/errors"
  "github.com/json-iterator/go"
  j "tezos-contests.izibi.com/backend/jase"
)

const (
  PageSize = 20
)

var HeadIndexExpiry = 1 * time.Hour
var PageIndexExpiry = 24 * time.Hour

/* Should it be keyed by game key, or last-block hash? */
func (st *Store) GetHeadIndex(gameKey string, lastBlock string) (uint64, []byte, error) {
  var err error
  var key = headIndexKey(gameKey)
  bs, _ := st.Redis.Get(key).Bytes()
  if len(bs) != 0 {
    type HeadIndex struct {
      Page uint64 `json:"page"`
      Blocks json.RawMessage `json:"blocks"`
    }
    var index HeadIndex
    err = json.Unmarshal(bs, &index)
    if err != nil { return 0, nil, errors.Wrap(err, 0) }
    return index.Page, []byte(index.Blocks), nil
  }
  page, blocks, err := st.buildHeadIndex(lastBlock)
  if err != nil { return 0, nil, err }
  var obj = j.Object()
  obj.Prop("page", j.Uint64(page))
  obj.Prop("blocks", j.Raw(blocks))
  objBytes, err := j.ToBytes(obj)
  if err != nil { return 0, nil, errors.Wrap(err, 0) }
  err = st.Redis.Set(key, objBytes, HeadIndexExpiry).Err()
  if err != nil { return 0, nil, errors.Wrap(err, 0) }
  return page, blocks, nil
}

func (st *Store) ClearHeadIndex(lastBlock string) error {
  err := st.Redis.Del(headIndexKey(lastBlock)).Err()
  if err != nil { return errors.Wrap(err, 0) }
  return nil
}

func (st *Store) GetPageIndex(gameKey string, lastBlock string, page uint64) ([]byte, error) {
  var err error
  /* Is the page index in the cache? */
  var pageKey = pageIndexKey(gameKey, page)
  bs, _ := st.Redis.Get(pageKey).Bytes()
  if len(bs) != 0 {
    return bs, nil
  }
  /* Starting with the HEAD index, use the page's first block as the parent of
     the last block of the preceding page, until we reach the requested page. */
  nextPage, blocks, err := st.GetHeadIndex(gameKey, lastBlock)
  if err != nil { return nil, err }
  if page >= nextPage {
    return nil, errors.New("bad page number")
  }
  for page < nextPage {
    nextPage--
    pageKey = pageIndexKey(gameKey, nextPage)
    cached, _ := st.Redis.Get(pageKey).Bytes()
    if len(bs) != 0 {
      blocks = cached
    } else {
      parentHash := jsoniter.Get(blocks, 0, "hash").ToString()
      blocks, err = st.buildPageIndex(parentHash)
      if err != nil { return nil, err }
      err = st.Redis.Set(pageKey, blocks, PageIndexExpiry).Err()
      if err != nil { return nil, err }
    }
  }
  return blocks, nil
}

func (st *Store) buildHeadIndex(lastBlock string) (uint64, []byte, error) {
  block, err := st.ReadBlock(lastBlock)
  if err != nil { return 0, nil, err }
  var base = block.Base()
  var page = base.Sequence / PageSize // truncated
  var nItems = int(base.Sequence - page * PageSize + 1)
  var hash = lastBlock
  // Follow parent relations, filling items in reverse order.
  var items = make([]j.Value, nItems, nItems)
  for i := nItems - 1; i >= 0; i-- {
    var item = j.Object()
    item.Prop("hash", j.String(hash))
    item.Prop("type", j.String(base.Kind))
    item.Prop("sequence", j.Uint64(base.Sequence))
    items[i] = item
    if base.Sequence == 0 {
      break
    }
    hash = base.Parent
    block, err := st.ReadBlock(hash)
    if err != nil {
      err = errors.Errorf("failed to load block %s: %v", hash, err)
      return 0, nil, err
    }
    base = block.Base()
  }
  // Build the index.
  var blocks = j.Array()
  for i := 0; i < len(items); i++ {
    blocks.Item(items[i])
  }
  bs, err := j.ToBytes(blocks)
  if err != nil { return 0, nil, errors.Wrap(err, 0) }
  return page, bs, nil
}

/* parentHash is the hash of the first block in the next page. */
func (st *Store) buildPageIndex(parentHash string) ([]byte, error) {
  block, err := st.ReadBlock(parentHash)
  if err != nil { return nil, err }
  var base = block.Base()
  var nextPage = base.Sequence / PageSize
  if nextPage == 0 { panic(fmt.Sprintf("buildPageIndex: bad parent %s", parentHash)) }
  var items = make([]j.Value, PageSize, PageSize)
  for i := PageSize - 1; i >= 0; i-- {
    hash := base.Parent
    block, err := st.ReadBlock(hash)
    if err != nil {
      err = errors.Errorf("failed to load block %s: %v", hash, err)
      return nil, err
    }
    base = block.Base()
    var item = j.Object()
    item.Prop("hash", j.String(hash))
    item.Prop("type", j.String(base.Kind))
    item.Prop("sequence", j.Uint64(base.Sequence))
    items[i] = item
  }
  // Build the index.
  var blocks = j.Array()
  for i := 0; i < len(items); i++ {
    blocks.Item(items[i])
  }
  bs, err := j.ToBytes(blocks)
  if err != nil { return nil, errors.Wrap(err, 0) }
  return bs, nil
}

func headIndexKey(gameKey string) string {
  return fmt.Sprintf("%s:HEAD", gameKey)
}

func pageIndexKey(gameKey string, page uint64) string {
  return fmt.Sprintf("%s:%d", gameKey, page)
}
