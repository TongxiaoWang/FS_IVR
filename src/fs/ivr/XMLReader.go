// fs/ivr/ XMLReader

package ivr

import (
	l4g "code.google.com/p/log4go"
	"encoding/xml"
	"fmt"
	"io/ioutil"
	"os"
)

type IVRConfig struct {
	Prompts  Prompts
	Grammars Grammars
	Nodes    Nodes
}

type Prompts struct {
	Prompt []Prompt
}

type Grammars struct {
	Grammar []Grammar
}

type Nodes struct {
	RootNode          RootNode
	ExitNode          ExitNode
	AnnNode           []AnnNode
	MenuNode          []MenuNode
	PromptCollectNode []PromptCollectNode
	GotoNode          []GotoNode
}

type Prompt struct {
	PName   string `xml:"name,attr"`
	BargeIn bool
	Phrase  []string
}

type PromptEntity struct {
	Prompt []string
}

type MenuChoice struct {
	Name     string `xml:"name,attr"`
	DTMF     string `xml:"dtmf,attr"`
	NextNode string `xml:"nextNode,attr"`
}

type Choices struct {
	Choice []MenuChoice
}

type Grammar struct {
	GName      string `xml:"name,attr"`
	MaxLen     int    // eg. "1","3~4"
	Terminator string
	Timeout    int
	Express    string
}

type GrammarEntity struct {
	Grammar []string
}

func LoadIVRConfig(name string) {

	fileInfo, err := os.Stat(name)
	if err != nil {
		l4g.Error("Load Ivr config file failure for : %s", err.Error())
		return
	}

	l4g.Debug("ConfigFile modTime=%d,loadTime=%d", fileInfo.ModTime(), ivr.confFileLoadTime)
	if fileInfo.ModTime().UnixNano() <= ivr.confFileLoadTime.UnixNano() {
		return
	}

	ivr.confFileLoadTime = fileInfo.ModTime()

	l4g.Trace("Init ivr config file from: %s", name)
	content, err := ioutil.ReadFile(name)
	if err != nil {
		l4g.Error("Load Ivr config file[%s] failure for %s", name, err.Error())
		return
	}

	// l4g.Debug("Load config content : %s", string(content))
	var ivrConfig IVRConfig
	err = xml.Unmarshal(content, &ivrConfig)
	if err != nil {
		l4g.Error("Unmarshal config xml failure for %s", err.Error())
		return
	}

	// fmt.Println(ivr)

	if len(ivrConfig.Prompts.Prompt) > 0 {
		for _, prompt := range ivrConfig.Prompts.Prompt {
			ivrPromptMap[prompt.PName] = prompt
		}
	} else {
		l4g.Warn("Init IVR config no prompt find ...")
	}

	if len(ivrConfig.Grammars.Grammar) > 0 {
		for _, grammar := range ivrConfig.Grammars.Grammar {
			ivrGrammarMap[grammar.GName] = grammar
		}
	} else {
		l4g.Warn("Init IVR config no grammar find ...")
	}

	if len(ivrConfig.Nodes.RootNode.NodeName) > 0 {
		ivrNodeMap[ivrConfig.Nodes.RootNode.NodeName] = ivrConfig.Nodes.RootNode
	} else {
		fmt.Println("No RootNode find ------------------------------------------------ ")
	}

	if len(ivrConfig.Nodes.ExitNode.NodeName) > 0 {
		ivrNodeMap[ivrConfig.Nodes.ExitNode.NodeName] = ivrConfig.Nodes.ExitNode
	} else {
		fmt.Println("No ExitNode find ------------------------------------------------ ")
	}

	if len(ivrConfig.Nodes.GotoNode) > 0 {
		for _, gotoNode := range ivrConfig.Nodes.GotoNode {
			ivrNodeMap[gotoNode.NodeName] = gotoNode
		}
	}

	if len(ivrConfig.Nodes.AnnNode) > 0 {
		for _, annNode := range ivrConfig.Nodes.AnnNode {
			ivrNodeMap[annNode.NodeName] = annNode
		}
	}

	if len(ivrConfig.Nodes.MenuNode) > 0 {
		for _, menuNode := range ivrConfig.Nodes.MenuNode {
			ivrNodeMap[menuNode.NodeName] = menuNode
		}
	}

	if len(ivrConfig.Nodes.PromptCollectNode) > 0 {
		for _, pcNode := range ivrConfig.Nodes.PromptCollectNode {
			ivrNodeMap[pcNode.NodeName] = pcNode
		}
	}

	l4g.Trace("Load ivrConfig prompts=%d,grammars=%d,nodes=%d", len(ivrPromptMap), len(ivrGrammarMap), len(ivrNodeMap))

}
