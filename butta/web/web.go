// Package web embute o frontend no binário do servidor.
//
// go:embed copia os arquivos pra DENTRO do executável em tempo de
// compilação — o deploy vira "sobe 1 binário", sem pasta de assets.
package web

import "embed"

//go:embed index.html
var FS embed.FS
