package astutils

import (
	"go/token"
	"sort"
)

// lists must contain only integers i.e > 0.
func mergeSortedListsUniq(aList, bList []int, max int) []int {
	lenA := len(aList)
	lenB := len(bList)

	sort.Ints(aList)
	sort.Ints(bList)

	result := make([]int, 0, lenA+lenB)

	a, b, prevValue := 0, 0, 0
	for a < lenA || b < lenB {
		value := 0

		// If within bounds of both lists.
		if a < lenA && b < lenB {
			if aList[a] > bList[b] {
				value = bList[b]
				b++
			} else if aList[a] < bList[b] {
				value = aList[a]
				a++
			} else { // Equality
				value = aList[a]
				a++
				b++
			}
		} else {
			if a >= lenA {
				value = bList[b]
				b++
			} else if b >= lenB {
				value = aList[a]
				a++
			}
		}

		// Ensure no duplicate elements.
		if value != prevValue && value < max {
			result = append(result, value)
			prevValue = value
		}
	}

	return result
}

func fileGetLines(f *token.File) []int {
	lines := make([]int, f.LineCount())
	for i := 0; i < f.LineCount(); i++ {
		lines[i] = f.Offset(f.LineStart(i + 1))
	}
	return lines
}
