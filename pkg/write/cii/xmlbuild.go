package cii

import (
	"strings"
)

// node est un nœud d'un petit arbre XML servant à générer du CII avec des préfixes de
// namespace explicites (rsm:, ram:, udt:). encoding/xml ne permet pas de contrôler
// finement les préfixes ; cet arbre le fait proprement.
type node struct {
	name     string // nom qualifié, ex. "ram:ID"
	attrs    []attr
	text     string // contenu textuel (feuille) ; ignoré si children non vide
	children []*node
}

type attr struct{ name, value string }

// el crée un élément conteneur.
func el(name string, children ...*node) *node {
	n := &node{name: name}
	for _, c := range children {
		if c != nil {
			n.children = append(n.children, c)
		}
	}
	return n
}

// leaf crée un élément feuille avec du texte. Renvoie nil si le texte est vide (pour élaguer
// les éléments optionnels non renseignés).
func leaf(name, text string) *node {
	if text == "" {
		return nil
	}
	return &node{name: name, text: text}
}

// leafA crée une feuille avec attributs (émise même si le texte est vide dès lors qu'un
// attribut est présent — utile pour DateTimeString@format).
func leafA(name, text string, attrs ...attr) *node {
	return &node{name: name, text: text, attrs: attrs}
}

// with ajoute des enfants à un nœud (les nil sont ignorés) et renvoie le nœud.
func (n *node) with(children ...*node) *node {
	for _, c := range children {
		if c != nil {
			n.children = append(n.children, c)
		}
	}
	return n
}

// render sérialise l'arbre avec indentation de deux espaces.
func render(root *node) string {
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	renderNode(&b, root, 0)
	return b.String()
}

func renderNode(b *strings.Builder, n *node, depth int) {
	indent := strings.Repeat("  ", depth)
	b.WriteString(indent)
	b.WriteByte('<')
	b.WriteString(n.name)
	for _, a := range n.attrs {
		b.WriteByte(' ')
		b.WriteString(a.name)
		b.WriteString(`="`)
		b.WriteString(escapeAttr(a.value))
		b.WriteByte('"')
	}
	if len(n.children) == 0 && n.text == "" {
		b.WriteString("/>\n")
		return
	}
	b.WriteByte('>')
	if len(n.children) == 0 {
		b.WriteString(escapeText(n.text))
		b.WriteString("</")
		b.WriteString(n.name)
		b.WriteString(">\n")
		return
	}
	b.WriteByte('\n')
	for _, c := range n.children {
		renderNode(b, c, depth+1)
	}
	b.WriteString(indent)
	b.WriteString("</")
	b.WriteString(n.name)
	b.WriteString(">\n")
}

func escapeText(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;")
	return r.Replace(s)
}

func escapeAttr(s string) string {
	r := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;")
	return r.Replace(s)
}
