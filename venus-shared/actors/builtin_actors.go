package actors

import (
	"archive/tar"
	"context"
	"embed"
	"fmt"
	"io"
	"os"
	"path"
	"sort"
	"strconv"
	"strings"

	"github.com/DataDog/zstd"
	actorstypes "github.com/filecoin-project/go-state-types/actors"
	"github.com/ipfs/go-cid"
	cbor "github.com/ipfs/go-ipld-cbor"
	"github.com/ipld/go-car"

	"github.com/filecoin-project/venus/venus-shared/actors/adt"
	blockstoreutil "github.com/filecoin-project/venus/venus-shared/blockstore"
)

//go:embed builtin-actors-code/*.tar.zst
var embeddedBuiltinActorReleases embed.FS

// NOTE: DO NOT change this unless you REALLY know what you're doing. This is consensus critical.
var BundleOverrides map[actorstypes.Version]string

var NetworkBundle = "mainnet"

func init() {
	if BundleOverrides == nil {
		BundleOverrides = make(map[actorstypes.Version]string)
	}

	for _, av := range Versions {
		path := os.Getenv(fmt.Sprintf("VENUS_BUILTIN_ACTORS_V%d_BUNDLE", av))
		if path == "" {
			continue
		}
		BundleOverrides[actorstypes.Version(av)] = path
	}
	if err := loadManifests(NetworkBundle); err != nil {
		panic(err)
	}
}

// NetworkMainnet    NetworkType = 0x1
// Network2k         NetworkType = 0x2
// NetworkCalibnet   NetworkType = 0x4
// NetworkInterop    NetworkType = 0x6
// NetworkForce      NetworkType = 0x7
// NetworkButterfly  NetworkType = 0x8
// Avoid import cycle, we use concrete values
func SetNetworkBundle(networkType int) error {
	networkBundle := ""
	switch networkType {
	// case types.Network2k:
	case 0x2:
		networkBundle = "devnet"
	// case types.NetworkCalibnet:
	case 0x4:
		networkBundle = "calibrationnet"
	// case types.NetworkInterop:
	case 0x6:
		networkBundle = "caterpillarnet"
	// types.NetworkForce
	case 0x7:
		networkBundle = "testing"
	// case types.NetworkButterfly:
	case 0x8:
		networkBundle = "butterflynet"
	default:
		networkBundle = "mainnet"
	}

	return UseNetworkBundle(networkBundle)
}

// UseNetworkBundle switches to a different network bundle, by name.
func UseNetworkBundle(netw string) error {
	if NetworkBundle == netw {
		return nil
	}
	NetworkBundle = netw

	return loadManifests(netw)
}

func loadManifests(netw string) error {
	overridden := make(map[actorstypes.Version]struct{})
	var newMetadata []*BuiltinActorsMetadata
	// First, prefer overrides.
	for av, path := range BundleOverrides {
		root, actorCids, err := readBundleManifestFromFile(path)
		if err != nil {
			return err
		}
		newMetadata = append(newMetadata, &BuiltinActorsMetadata{
			Network:     netw,
			Version:     av,
			ManifestCid: root,
			Actors:      actorCids,
		})
		overridden[av] = struct{}{}
	}

	// Then load embedded bundle metadata.
	for _, meta := range EmbeddedBuiltinActorsMetadata {
		if meta.Network != netw {
			continue
		}
		if _, ok := overridden[meta.Version]; ok {
			continue
		}
		newMetadata = append(newMetadata, meta)
	}

	ClearManifests()

	for _, meta := range newMetadata {
		RegisterManifest(meta.Version, meta.ManifestCid, meta.Actors)
	}

	// The following code cid existed temporarily on the calibnet testnet, as a "buggy" storage miner actor implementation.
	// We include them in our builtin bundle, but intentionally omit from metadata.
	if NetworkBundle == "calibrationnet" {
		AddActorMeta("storageminer", cid.MustParse("bafk2bzacecnh2ouohmonvebq7uughh4h3ppmg4cjsk74dzxlbbtlcij4xbzxq"), actorstypes.Version12)
		AddActorMeta("storageminer", cid.MustParse("bafk2bzaced7emkbbnrewv5uvrokxpf5tlm4jslu2jsv77ofw2yqdglg657uie"), actorstypes.Version12)
		AddActorMeta("verifiedregistry", cid.MustParse("bafk2bzacednskl3bykz5qpo54z2j2p4q44t5of4ktd6vs6ymmg2zebsbxazkm"), actorstypes.Version13)
	}

	return nil
}

type BuiltinActorsMetadata struct { // nolint
	Network      string
	Version      actorstypes.Version
	ManifestCid  cid.Cid
	Actors       map[string]cid.Cid
	BundleGitTag string
}

// ReadEmbeddedBuiltinActorsMetadata reads the metadata from the embedded built-in actor bundles.
// There should be no need to call this method as the result is cached in the
// `EmbeddedBuiltinActorsMetadata` variable on `make gen`.
func ReadEmbeddedBuiltinActorsMetadata() ([]*BuiltinActorsMetadata, error) {
	files, err := embeddedBuiltinActorReleases.ReadDir("builtin-actors-code")
	if err != nil {
		return nil, fmt.Errorf("failed to read embedded bundle directory: %s", err)
	}
	var bundles []*BuiltinActorsMetadata
	for _, dirent := range files {
		name := dirent.Name()
		b, err := readEmbeddedBuiltinActorsMetadata(name)
		if err != nil {
			return nil, err
		}
		bundles = append(bundles, b...)
	}
	// Sort by network, then by bundle.
	sort.Slice(bundles, func(i, j int) bool {
		if bundles[i].Network == bundles[j].Network {
			return bundles[i].Version < bundles[j].Version
		}
		return bundles[i].Network < bundles[j].Network
	})
	return bundles, nil
}

func readEmbeddedBuiltinActorsMetadata(bundle string) ([]*BuiltinActorsMetadata, error) {
	const (
		archiveExt   = ".tar.zst"
		bundleExt    = ".car"
		bundlePrefix = "builtin-actors-"
	)

	if !strings.HasPrefix(bundle, "v") {
		return nil, fmt.Errorf("bundle bundle '%q' doesn't start with a 'v'", bundle)
	}
	if !strings.HasSuffix(bundle, archiveExt) {
		return nil, fmt.Errorf("bundle bundle '%q' doesn't end with '%s'", bundle, archiveExt)
	}
	version, err := strconv.ParseInt(bundle[1:len(bundle)-len(archiveExt)], 10, 0)
	if err != nil {
		return nil, fmt.Errorf("failed to parse actors version from bundle '%q': %s", bundle, err)
	}
	fi, err := embeddedBuiltinActorReleases.Open(fmt.Sprintf("builtin-actors-code/%s", bundle))
	if err != nil {
		return nil, err
	}
	defer fi.Close() //nolint

	uncompressed := zstd.NewReader(fi)
	defer uncompressed.Close() //nolint

	var bundles []*BuiltinActorsMetadata

	tarReader := tar.NewReader(uncompressed)
	for {
		header, err := tarReader.Next()
		switch err {
		case io.EOF:
			return bundles, nil
		case nil:
		default:
			return nil, err
		}

		// Read the network name from the bundle name.
		name := path.Base(header.Name)
		if !strings.HasSuffix(name, bundleExt) {
			return nil, fmt.Errorf("expected bundle to end with .car: %s", name)
		}
		if !strings.HasPrefix(name, bundlePrefix) {
			return nil, fmt.Errorf("expected bundle to end with .car: %s", name)
		}
		name = name[len(bundlePrefix) : len(name)-len(bundleExt)]

		// Load the bundle.
		root, actorCids, err := readBundleManifest(tarReader)
		if err != nil {
			return nil, fmt.Errorf("error loading builtin actors bundle: %w", err)
		}

		// The following manifest cids existed temporarily on the calibnet testnet
		// We include them in our builtin bundle, but intentionally omit from metadata
		if root == cid.MustParse("bafy2bzacedrunxfqta5skb7q7x32lnp4efz2oq7fn226ffm7fu5iqs62jkmvs") ||
			root == cid.MustParse("bafy2bzacebl4w5ptfvuw6746w7ev562idkbf5ppq72e6zub22435ws2rukzru") ||
			root == cid.MustParse("bafy2bzacea4firkyvt2zzdwqjrws5pyeluaesh6uaid246tommayr4337xpmi") {
			continue
		}
		bundles = append(bundles, &BuiltinActorsMetadata{
			Network:     name,
			Version:     actorstypes.Version(version),
			ManifestCid: root,
			Actors:      actorCids,
		})
	}
}

func readBundleManifestFromFile(path string) (cid.Cid, map[string]cid.Cid, error) {
	fi, err := os.Open(path)
	if err != nil {
		return cid.Undef, nil, err
	}
	defer fi.Close() //nolint

	return readBundleManifest(fi)
}

func readBundleManifest(r io.Reader) (cid.Cid, map[string]cid.Cid, error) {
	// Load the bundle.
	bs := blockstoreutil.NewMemory()
	hdr, err := car.LoadCar(context.Background(), bs, r)
	if err != nil {
		return cid.Undef, nil, fmt.Errorf("error loading builtin actors bundle: %w", err)
	}

	if len(hdr.Roots) != 1 {
		return cid.Undef, nil, fmt.Errorf("expected one root when loading actors bundle, got %d", len(hdr.Roots))
	}
	root := hdr.Roots[0]
	actorCids, err := ReadManifest(context.Background(), adt.WrapStore(context.Background(), cbor.NewCborStore(bs)), root)
	if err != nil {
		return cid.Undef, nil, err
	}

	// Make sure we have all the
	for name, c := range actorCids {
		if has, err := bs.Has(context.Background(), c); err != nil {
			return cid.Undef, nil, fmt.Errorf("got an error when checking that the bundle has the actor %q: %w", name, err)
		} else if !has {
			return cid.Undef, nil, fmt.Errorf("actor %q missing from bundle", name)
		}
	}

	return root, actorCids, nil
}

// GetEmbeddedBuiltinActorsBundle returns the builtin-actors bundle for the given actors version.
func GetEmbeddedBuiltinActorsBundle(version actorstypes.Version, networkBundleName string) ([]byte, bool) {
	fi, err := embeddedBuiltinActorReleases.Open(fmt.Sprintf("builtin-actors-code/v%d.tar.zst", version))
	if err != nil {
		return nil, false
	}
	defer fi.Close() //nolint

	uncompressed := zstd.NewReader(fi)
	defer uncompressed.Close() //nolint

	tarReader := tar.NewReader(uncompressed)
	targetFileName := fmt.Sprintf("builtin-actors-%s.car", networkBundleName)
	for {
		header, err := tarReader.Next()
		switch err {
		case io.EOF:
			return nil, false
		case nil:
		default:
			panic(err)
		}
		if header.Name != targetFileName {
			continue
		}

		car, err := io.ReadAll(tarReader)
		if err != nil {
			panic(err)
		}
		return car, true
	}
}
