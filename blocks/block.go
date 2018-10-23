
package blocks

type Block interface {
  Base() *BlockBase
}

func LastSetupBlock(hash string, block Block) string {
  base := block.Base()
  if base.Kind == "setup" {
    return hash
  }
  return base.Setup
}
