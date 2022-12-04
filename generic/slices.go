package generic

func Filter[T any](arr []T, filterFunc func(T) bool) []T {

	if arr == nil {
		return nil
	}

	filteredArr := make([]T, 0)

	for i := range arr {
		if filterFunc(arr[i]) {
			filteredArr = append(filteredArr, arr[i])
		}
	}

	return filteredArr
}

func Map[T any, U any](arr []T, mapFunc func(T) U) []U {

	if arr == nil {
		return nil
	}

	mapArr := make([]U, len(arr))

	for i := range arr {
		mapArr[i] = mapFunc(arr[i])
	}

	return mapArr
}
