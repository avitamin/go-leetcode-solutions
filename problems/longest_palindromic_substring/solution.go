package longest_palindromic_substring

func longestPalindrome(s string) string {
	if len(s) < 2 {
		return s
	}

	bestStart, bestEnd := 0, 0

	for center := 0; center < len(s); center++ {
		oddStart, oddEnd := expandAroundCenter(s, center, center)
		if oddEnd-oddStart > bestEnd-bestStart {
			bestStart, bestEnd = oddStart, oddEnd
		}

		evenStart, evenEnd := expandAroundCenter(s, center, center+1)
		if evenEnd-evenStart > bestEnd-bestStart {
			bestStart, bestEnd = evenStart, evenEnd
		}
	}

	return s[bestStart : bestEnd+1]
}

func expandAroundCenter(s string, left, right int) (int, int) {
	for left >= 0 && right < len(s) && s[left] == s[right] {
		left--
		right++
	}

	return left + 1, right - 1
}
