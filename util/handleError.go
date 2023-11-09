package util

func HandleError(err error) {
	if err != nil {
		println("ERROR:", err)
	}
}
