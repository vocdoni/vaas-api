package notify

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"reflect"
	"strings"
	"sync"
	"time"

	"gitlab.com/vocdoni/go-dvote/chain"
	"gitlab.com/vocdoni/go-dvote/config"
	"gitlab.com/vocdoni/go-dvote/data"
	"gitlab.com/vocdoni/go-dvote/log"
	"gitlab.com/vocdoni/go-dvote/metrics"
	"gitlab.com/vocdoni/manager/manager-backend/database"
	"gitlab.com/vocdoni/manager/manager-backend/types"
)

// The IPFSFileTracker is in charge of tracking IPFS files and
// notify if the content of any file was changed.
// We can obtain the list of files to track resolving each entity
// metadata using the ENS system. The list of entities is obtained
// from the manager database.

// In short it has two main loops: One for refreshing the entities list
// and another for fetching files from IPFS and update them if any change.
// If the file was updated then the content and relevant information
// is sended through the UpdatedFilesQueue channel in order to be handled
// by the push notifications service.

// RetrieveTimeout the maximum duration the import queue will wait for retreiving a remote file
const RetrieveTimeout = 1 * time.Minute

// IPFSFile holds the ipfs hash of the entity metadata and the relevant content
type IPFSFile struct {
	Hash    string
	Content map[string]string // TODO: @jordipainan Use struct once defined
}

// UpdatedFile wraps the IPFS updated file and the creator entity
type UpdatedFile struct {
	*IPFSFile
	eID string
}

// IPFSFileTracker contains all the components of a IPFSFileTracker
type IPFSFileTracker struct {
	IPFS                   *data.IPFSHandle
	IPFSConfig             *config.IPFSCfg
	FileContentList        *sync.Map // [eID]IPFSFile
	UpdatedFilesQueue      chan *UpdatedFile
	EntitiesTrackingStatus *sync.Map
	database               database.Database
	ensRegistryAddress     string
	metricsAgent           *metrics.Agent
	w3endpoint             string
}

// NewIPFSFileTracker creates a new IPFSFileTracker
func NewIPFSFileTracker(config *config.IPFSCfg, ma *metrics.Agent, db database.Database, ensRegistry, w3endpoint string) *IPFSFileTracker {
	return &IPFSFileTracker{
		IPFS:                   new(data.IPFSHandle),
		IPFSConfig:             config,
		FileContentList:        new(sync.Map),
		UpdatedFilesQueue:      make(chan *UpdatedFile),
		EntitiesTrackingStatus: new(sync.Map),
		metricsAgent:           ma,
		database:               db,
		ensRegistryAddress:     ensRegistry,
		w3endpoint:             w3endpoint,
	}
}

// Start initializes the file tracker IPFS node and starts the file tracker
func (ft *IPFSFileTracker) Start(ctx context.Context) error {
	// init IPFS node
	storage, err := ft.initIPFS()
	if err != nil {
		return err
	}
	ft.IPFS = storage.(*data.IPFSHandle)
	// init file tracker
	go ft.refreshLoop(ctx)
	return err
}

func (ft *IPFSFileTracker) initIPFS() (data.Storage, error) {
	ctx := context.Background()
	log.Info("creating ipfs service")
	var storage data.Storage
	var err error
	if !ft.IPFSConfig.NoInit {
		os.Setenv("IPFS_FD_MAX", "1024")
		ipfsStore := data.IPFSNewConfig(ft.IPFSConfig.ConfigPath)
		storage, err = data.Init(data.StorageIDFromString("IPFS"), ipfsStore)
		if err != nil {
			return nil, err
		}

		go func() {
			for {
				time.Sleep(time.Second * 20)
				stats, err := storage.Stats(ctx)
				if err != nil {
					log.Warnf("IPFS node returned an error: %s", err)
				}
				log.Infof("[ipfs info] %s", stats)
			}
		}()

		go storage.CollectMetrics(ft.metricsAgent, ctx)
	}
	return storage, nil
}

func (ft *IPFSFileTracker) getEntities() ([]string, error) {
	entities, err := ft.database.EntitiesID()
	if err != nil {
		return nil, err
	}
	return entities, nil
}

func (ft *IPFSFileTracker) getEntityMetadataURL(eID string) (string, error) {
	return chain.ResolveEntityMetadataURL(ft.ensRegistryAddress, eID, ft.w3endpoint)
}

func (ft *IPFSFileTracker) refreshEntities(ctx context.Context) error {
	_, cancel := context.WithTimeout(ctx, RetrieveTimeout)
	defer cancel()
	// get entities to track
	eIDs, err := ft.getEntities()
	if err != nil {
		return err
	}
	updatedList := []string{}
	for _, e := range eIDs {
		updatedList = append(updatedList, e)
		ft.FileContentList.LoadOrStore(e, nil)
	}
	log.Debugf("updated entity list: %+v", updatedList)
	return nil
}

func (ft *IPFSFileTracker) refreshFileContent(ctx context.Context, key string) error {
	defer ft.EntitiesTrackingStatus.Store(key, false)
	// retrieve new metadata URL
	eURL, err := ft.getEntityMetadataURL(key)
	if err != nil {
		// return error
		return err
		// tracking status false
		// continue with the next key on range
	}
	// check eURL
	if len(eURL) < 2 {
		return errors.New("invalid entity metadata URL length")
	}
	log.Debugf("fetched entity %s metadata url %s", key, eURL)
	// get file
	contentBytes, err := ft.IPFS.Retrieve(ctx, eURL)
	if err != nil {
		return err
	}
	ipfsHash := strings.TrimPrefix(strings.Split(eURL, ",")[0], "ipfs://")
	// unmarshal retrived file
	var entityMetadata types.EntityMetadata
	err = json.Unmarshal(contentBytes, &entityMetadata)
	if err != nil {
		return err
	}
	log.Debugf("entity %s metadata is: %+v", key, entityMetadata)
	// compare current and fetched hash
	// load old content from FileContentList
	oldContent, _ := ft.FileContentList.Load(key)
	uFile := &UpdatedFile{eID: key, IPFSFile: &IPFSFile{Hash: ipfsHash, Content: entityMetadata.NewsFeed}}
	// if old content exists
	if oldContent != nil {
		oldContentStruct := oldContent.(IPFSFile)
		// check if different hash
		if oldContentStruct.Hash != ipfsHash {
			// check if the retrieved news feed is equal to the old news feed
			sameFeed := reflect.DeepEqual(entityMetadata.NewsFeed, oldContentStruct.Content)
			if !sameFeed {
				// TODO: @jordipainan, return entityID + exact feed
				// notify updated entity newsFeed
				ft.UpdatedFilesQueue <- uFile
				// delete old feed
				ft.FileContentList.Delete(key)
				ft.FileContentList.Store(uFile.eID, *uFile.IPFSFile)
				log.Debugf("entity %s metadata updated, hash: %s content: %+v", uFile.eID, uFile.Hash, *uFile.IPFSFile)
			}
		}
		// if same hash, nothing to do
	} else { // if not exists notify and store
		ft.UpdatedFilesQueue <- uFile
		ft.FileContentList.Store(uFile.eID, *uFile.IPFSFile)
		log.Debugf("entity %s metadata stored for first time, hash: %s file: %+v", uFile.eID, uFile.Hash, *uFile.IPFSFile)
	}
	return nil
}

func (ft *IPFSFileTracker) refreshFileContentList(ctx context.Context) []error {
	// init waitgroup and counter
	wg := new(sync.WaitGroup)
	errorList := []error{}
	var errListMu sync.Mutex
	timeout, cancel := context.WithTimeout(ctx, RetrieveTimeout)
	defer cancel()
	// iterate over the fileContentList
	// skip if the file is already tracked by another goroutine
	// else create a go routine and start to look for news feed changes
	// each go routine will terminate if:
	// 	- error
	//	- timeout
	//	- success
	// passing then to the next value of the FileContentList
	ft.FileContentList.Range(func(key, value interface{}) bool {
		isTracked, found := ft.EntitiesTrackingStatus.Load(key)
		// if already refreshing
		if found {
			if isTracked != nil && isTracked.(bool) {
				return true
			}
		}
		ft.EntitiesTrackingStatus.Store(key, true)
		// add to wait group
		wg.Add(1)
		// exec refresh goroutine for each file
		log.Debugf("refresing entity %s metadata", key.(string))
		go func() {
			if err := ft.refreshFileContent(timeout, key.(string)); err != nil {
				errListMu.Lock()
				errorList = append(errorList, err)
				errListMu.Unlock()
			}
			wg.Done()
		}()
		// iterate until end
		return true
	})
	// wait all goroutines to finish. All the goroutines will always be finished once
	// the RetrieveTimeout is reached or executed successfully.
	wg.Wait()
	// finish tracking round
	return errorList
}

func (ft IPFSFileTracker) refreshLoop(ctx context.Context) {
	for ctx.Err() == nil {
		log.Debug("refresh loop has finished, starting new iteration")
		log.Info("refreshing entities ...")
		if err := ft.refreshEntities(ctx); err != nil {
			log.Infof("cannot refresh entities, error: %v", err)
		} else {
			log.Info("entities updated")
		}
		log.Info("refreshing file content list ...")
		refreshFilesErrs := ft.refreshFileContentList(ctx)
		if len(refreshFilesErrs) > 0 {
			log.Infof("entities files refresh error list: %+v", refreshFilesErrs)
		} else {
			log.Info("all files updated successfully")
		}
		time.Sleep(time.Second * 10)
	}
}
