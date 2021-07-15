package main

func uniqueArrayOfValues(arr []string) (unique []string) {
	set := make(map[string]struct{})
	for _, val := range arr {
		if _, ok := set[val]; !ok {
			set[val] = struct{}{}
			unique = append(unique, val)
		}
	}
	return
}
