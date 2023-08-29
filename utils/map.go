package utils

func MapCopy(originalMap map[string]interface{}) map[string]interface{} {
	if len(originalMap) == 0 {
		return originalMap
	}
	targetMap := make(map[string]interface{}, len(originalMap))

	for key, value := range originalMap {
		switch v := value.(type) {
		case map[string]interface{}:
			value = MapCopy(v)
			//todo other type,such as array/pointer...
		}
		targetMap[key] = value
	}
	return targetMap
}
