package helper

// const (
// 	regionCollName = "regions"
// )

// var (
// 	regionColl *mongo.Collection
// 	cache      = map[string]models.RegionInfo{}
// 	rwMutex    sync.RWMutex
// )

// func GetRegionInfo(ctx context.Context, rg models.IRegion) (models.RegionInfo, error) {
// 	rwMutex.RLock()
// 	info, exist := cache[rg.GetRegion()]
// 	rwMutex.RUnlock()
// 	if exist {
// 		return info, nil
// 	}
// 	rwMutex.Lock()
// 	defer rwMutex.Unlock()

// 	if regionColl == nil {
// 		regionColl = db.NewCollection(regionCollName)
// 	}

// 	result := regionColl.FindOne(ctx, bson.M{"name": rg.GetRegion()})
// 	info = models.RegionInfo{}
// 	if err := result.Decode(&info); err != nil {
// 		return models.RegionInfo{}, err
// 	}
// 	// TODO: how to refresh cache?
// 	cache[rg.GetRegion()] = info
// 	return info, nil
// }

// func GetAllRegionInfo(ctx context.Context) ([]models.RegionInfo, error) {
// 	rwMutex.RLock()
// 	infos := make([]models.RegionInfo, 0)
// 	for _, region := range cache {
// 		infos = append(infos, region)
// 	}
// 	rwMutex.RUnlock()

// 	if len(infos) != 0 {
// 		return infos, nil
// 	}

// 	rwMutex.Lock()
// 	defer rwMutex.Unlock()
// 	if regionColl == nil {
// 		regionColl = db.NewCollection(regionCollName)
// 	}

// 	cursor, err := regionColl.Find(ctx, bson.M{})
// 	if err != nil {
// 		return nil, err
// 	}
// 	defer func() {
// 		_ = cursor.Close(ctx)
// 	}()

// 	for cursor.Next(ctx) {
// 		info := models.RegionInfo{}
// 		err = cursor.Decode(&info)
// 		if err != nil {
// 			return nil, err
// 		}
// 		cache[models.Region(info.Name)] = info
// 		infos = append(infos, info)
// 	}
// 	return infos, nil
// }
