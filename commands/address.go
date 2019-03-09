package commands

import (
	"encoding/base64"
	"fmt"
	"io"

	"github.com/ipfs/go-ipfs-cmdkit"
	"github.com/ipfs/go-ipfs-cmds"
	"github.com/ipfs/go-ipfs-files"

	"github.com/filecoin-project/go-filecoin/address"
	"github.com/filecoin-project/go-filecoin/api/impl"
	"github.com/filecoin-project/go-filecoin/types"
)

var walletCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Manage your filecoin wallets",
	},
	Subcommands: map[string]*cmds.Command{
		"addrs":   addrsCmd,
		"balance": balanceCmd,
		"import":  walletImportCmd,
		"export":  walletExportCmd,
	},
}

var addrsCmd = &cmds.Command{
	Helptext: cmdkit.HelpText{
		Tagline: "Interact with addresses",
	},
	Subcommands: map[string]*cmds.Command{
		"ls":     addrsLsCmd,
		"new":    addrsNewCmd,
		"lookup": addrsLookupCmd,
	},
}

type addressResult struct {
	Address string
}

// AddressLsResult is the result of running the address list command.
type AddressLsResult struct {
	Addresses []string
}

var addrsNewCmd = &cmds.Command{
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) error {
		addr, err := GetAPI(env).Address().Addrs().New(req.Context)
		if err != nil {
			return err
		}
		return re.Emit(&addressResult{addr.String()})
	},
	Type: &addressResult{},
	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeTypedEncoder(func(req *cmds.Request, w io.Writer, a *addressResult) error {
			_, err := fmt.Fprintln(w, a.Address)
			return err
		}),
	},
}

var addrsLsCmd = &cmds.Command{
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) error {
		addrs, err := GetAPI(env).Address().Addrs().Ls(req.Context)
		if err != nil {
			return err
		}

		var alr AddressLsResult
		for _, addr := range addrs {
			alr.Addresses = append(alr.Addresses, addr.String())
		}

		return re.Emit(&alr)
	},
	Type: &AddressLsResult{},
	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeTypedEncoder(func(req *cmds.Request, w io.Writer, addrs *AddressLsResult) error {
			for _, addr := range addrs.Addresses {
				_, err := fmt.Fprintln(w, addr)
				if err != nil {
					return err
				}
			}
			return nil
		}),
	},
}

var addrsLookupCmd = &cmds.Command{
	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("address", true, false, "Miner address to find peerId for"),
	},
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) error {
		addr, err := address.NewFromString(req.Arguments[0])
		if err != nil {
			return err
		}

		v, err := GetAPI(env).Address().Addrs().Lookup(req.Context, addr)
		if err != nil {
			return err
		}
		return re.Emit(v.Pretty())
	},
	Type: string(""),
	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeTypedEncoder(func(req *cmds.Request, w io.Writer, pid string) error {
			_, err := fmt.Fprintln(w, pid)
			return err
		}),
	},
}

var balanceCmd = &cmds.Command{
	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("address", true, false, "Address to get balance for"),
	},
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) error {
		addr, err := address.NewFromString(req.Arguments[0])
		if err != nil {
			return err
		}

		balance, err := GetPorcelainAPI(env).WalletBalance(req.Context, addr)
		if err != nil {
			return err
		}
		return re.Emit(balance)
	},
	Type: &types.AttoFIL{},
	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeTypedEncoder(func(req *cmds.Request, w io.Writer, b *types.AttoFIL) error {
			return PrintString(w, b)
		}),
	},
}

var walletImportCmd = &cmds.Command{
	Arguments: []cmdkit.Argument{
		cmdkit.FileArg("walletFile", true, false, "File containing wallet data to import").EnableStdin(),
	},
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) error {
		iter := req.Files.Entries()
		if !iter.Next() {
			return fmt.Errorf("no file given: %s", iter.Err())
		}

		fi, ok := iter.Node().(files.File)
		if !ok {
			return fmt.Errorf("given file was not a files.File")
		}

		addrs, err := GetAPI(env).Address().Import(req.Context, fi)
		if err != nil {
			return err
		}

		var alr AddressLsResult
		for _, addr := range addrs {
			alr.Addresses = append(alr.Addresses, addr.String())
		}

		return re.Emit(&alr)
	},
	Type: &AddressLsResult{},
	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeTypedEncoder(func(req *cmds.Request, w io.Writer, addrs *AddressLsResult) error {
			for _, addr := range addrs.Addresses {
				_, err := fmt.Fprintln(w, addr)
				if err != nil {
					return err
				}
			}
			return nil
		}),
	},
}

var walletExportCmd = &cmds.Command{
	Arguments: []cmdkit.Argument{
		cmdkit.StringArg("addresses", true, true, "Addresses of keys to export").EnableStdin(),
	},
	Run: func(req *cmds.Request, re cmds.ResponseEmitter, env cmds.Environment) error {
		addrs := make([]address.Address, len(req.Arguments))
		for i, arg := range req.Arguments {
			addr, err := address.NewFromString(arg)
			if err != nil {
				return err
			}
			addrs[i] = addr
		}

		kis, err := GetAPI(env).Address().Export(req.Context, addrs)
		if err != nil {
			return err
		}

		var klr impl.WalletSerializeResult
		klr.KeyInfo = append(klr.KeyInfo, kis...)

		return re.Emit(klr)
	},
	Type: &impl.WalletSerializeResult{},
	Encoders: cmds.EncoderMap{
		cmds.Text: cmds.MakeTypedEncoder(func(req *cmds.Request, w io.Writer, klr *impl.WalletSerializeResult) error {
			for _, k := range klr.KeyInfo {
				a, err := k.Address()
				if err != nil {
					return err
				}
				privateKeyInBase64 := base64.StdEncoding.EncodeToString(k.PrivateKey)
				_, err = fmt.Fprintf(w, "Address:\t%s\nPrivateKey:\t%s\nCurve:\t\t%s\n\n", a.String(), privateKeyInBase64, k.Curve)
				if err != nil {
					return err
				}
			}
			return nil
		}),
	},
}
