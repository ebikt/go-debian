/* {{{ Copyright (c) Paul R. Tagliamonte <paultag@debian.org>, 2015
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in
 * all copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
 * THE SOFTWARE. }}} */

package dependency_test

import (
	"log"
	"testing"

	"pault.ag/x/go-debian/dependency"
)

/*
 *
 */

func isok(t *testing.T, err error) {
	if err != nil {
		log.Printf("Error! Error is not nil! %s\n", err)
		t.FailNow()
	}
}

func notok(t *testing.T, err error) {
	if err == nil {
		log.Printf("Error! Error is nil!\n")
		t.FailNow()
	}
}

func assert(t *testing.T, expr bool) {
	if !expr {
		log.Printf("Assertion failed!")
		t.FailNow()
	}
}

/*
 *
 */

func TestSingleParse(t *testing.T) {
	dep, err := dependency.Parse("foo")
	isok(t, err)

	if dep.Relations[0].Possibilities[0].Name != "foo" {
		t.Fail()
	}
}

func TestMultiarchParse(t *testing.T) {
	dep, err := dependency.Parse("foo:amd64")
	isok(t, err)

	assert(t, dep.Relations[0].Possibilities[0].Name == "foo")
	assert(t, dep.Relations[0].Possibilities[0].Arch.CPU == "amd64")

	dep, err = dependency.Parse("foo:amd64 [amd64 sparc]")
	isok(t, err)

	assert(t, dep.Relations[0].Possibilities[0].Name == "foo")
	assert(t, dep.Relations[0].Possibilities[0].Arch.CPU == "amd64")

	assert(t, dep.Relations[0].Possibilities[0].Architectures.Architectures[0].CPU == "amd64")
	assert(t, dep.Relations[0].Possibilities[0].Architectures.Architectures[1].CPU == "sparc")
}

func TestTwoRelations(t *testing.T) {
	dep, err := dependency.Parse("foo, bar")
	isok(t, err)
	assert(t, len(dep.Relations) == 2)
}

func TestTwoPossibilities(t *testing.T) {
	dep, err := dependency.Parse("foo, bar | baz")
	isok(t, err)
	assert(t, len(dep.Relations) == 2)

	possi := dep.Relations[1].Possibilities
	assert(t, len(possi) == 2)

	assert(t, possi[0].Name == "bar")
	assert(t, possi[1].Name == "baz")
}

func TestVersioning(t *testing.T) {
	dep, err := dependency.Parse("foo (>= 1.0)")
	isok(t, err)
	assert(t, len(dep.Relations) == 1)

	possi := dep.Relations[0].Possibilities[0]
	version := possi.Version

	assert(t, version.Operator == ">=")
	assert(t, version.Number == "1.0")
}

func TestSingleArch(t *testing.T) {
	dep, err := dependency.Parse("foo [arch]")
	isok(t, err)
	assert(t, len(dep.Relations) == 1)

	possi := dep.Relations[0].Possibilities[0]
	arches := possi.Architectures.Architectures

	assert(t, len(arches) == 1)
	assert(t, arches[0].CPU == "arch")
}

func TestSingleNotArch(t *testing.T) {
	dep, err := dependency.Parse("foo [!arch]")
	isok(t, err)
	assert(t, len(dep.Relations) == 1)

	possi := dep.Relations[0].Possibilities[0]
	arches := possi.Architectures.Architectures

	assert(t, len(arches) == 1)
	assert(t, arches[0].CPU == "arch")
	assert(t, possi.Architectures.Not)
}

func TestDoubleInvalidNotArch(t *testing.T) {
	_, err := dependency.Parse("foo [arch !foo]")
	notok(t, err)

	_, err = dependency.Parse("foo [arch!foo]")
	notok(t, err)
}

func TestDoubleArch(t *testing.T) {
	dep, err := dependency.Parse("foo [arch arch2]")
	isok(t, err)
	assert(t, len(dep.Relations) == 1)

	possi := dep.Relations[0].Possibilities[0]
	arches := possi.Architectures.Architectures

	assert(t, len(arches) == 2)
	assert(t, arches[0].CPU == "arch")
	assert(t, arches[1].CPU == "arch2")
}

func TestVersioningOperators(t *testing.T) {
	opers := map[string]string{
		">=": "foo (>= 1.0)",
		"<=": "foo (<= 1.0)",
		">>": "foo (>> 1.0)",
		"<<": "foo (<< 1.0)",
		"=":  "foo (= 1.0)",
	}

	for operator, vstring := range opers {
		dep, err := dependency.Parse(vstring)
		isok(t, err)
		assert(t, len(dep.Relations) == 1)
		possi := dep.Relations[0].Possibilities[0]
		version := possi.Version
		assert(t, version.Operator == operator)
		assert(t, version.Number == "1.0")
	}
}

func TestNoComma(t *testing.T) {
	_, err := dependency.Parse("foo bar")
	notok(t, err)
}

func TestTwoVersions(t *testing.T) {
	_, err := dependency.Parse("foo (>= 1.0) (<= 2.0)")
	notok(t, err)
}

func TestTwoArchitectures(t *testing.T) {
	_, err := dependency.Parse("foo [amd64] [sparc]")
	notok(t, err)
}

func TestBadVersion(t *testing.T) {
	vers := []string{
		"foo (>= 1.0",
		"foo (>= 1",
		"foo (>= ",
		"foo (>=",
		"foo (>",
		"foo (",
	}

	for _, ver := range vers {
		_, err := dependency.Parse(ver)
		notok(t, err)
	}
}

func TestBadArch(t *testing.T) {
	vers := []string{
		"foo [amd64",
		"foo [amd6",
		"foo [amd",
		"foo [am",
		"foo [a",
		"foo [",
	}

	for _, ver := range vers {
		_, err := dependency.Parse(ver)
		notok(t, err)
	}
}

func TestSingleSubstvar(t *testing.T) {
	dep, err := dependency.Parse("${foo:Depends}, bar, baz")
	isok(t, err)
	assert(t, len(dep.Relations) == 3)

	assert(t, dep.Relations[0].Possibilities[0].Name == "foo:Depends")
	assert(t, dep.Relations[1].Possibilities[0].Name == "bar")
	assert(t, dep.Relations[2].Possibilities[0].Name == "baz")

	assert(t, dep.Relations[0].Possibilities[0].Substvar)

	assert(t, !dep.Relations[1].Possibilities[0].Substvar)
	assert(t, !dep.Relations[2].Possibilities[0].Substvar)
}

// vim: foldmethod=marker
