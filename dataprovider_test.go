package vfs

func QueryExample_query(dp DataProvider) error {
	cursor, err := dp.Query(NewQuery().Select().MatchPath(Path("/asd/")).MatchParent(Path("/any/other")))
	if err != nil {
		return err
	}
	defer cursor.Close()
	//we can allocate our info object outside
	tmp := &ResourceInfo{}
	err = cursor.ForEach(func(reader AttributesReader) (next bool, err error) {
		err = reader.Attributes(tmp)
		if err != nil {
			return
		}
		next = true
		return
	})
	return err
}
