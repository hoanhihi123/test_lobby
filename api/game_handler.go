package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"sync"

	"github.com/nakamaFramework/cgb-lobby-module/api/presenter"
	"github.com/nakamaFramework/cgb-lobby-module/cgbdb"
	"github.com/nakamaFramework/cgb-lobby-module/entity"
	"github.com/nakamaFramework/cgp-common/define"
	pb "github.com/nakamaFramework/cgp-common/proto"

	"github.com/heroiclabs/nakama-common/runtime"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	kLobbyCollection = "lobby"
	kGameKey         = "games"
)

// var mapGameByCode = make(map[string]entity.Game, 0)
// var mapBetsByGameCode = make(map[string][]entity.Bet, 0)
var mapGameByCode sync.Map     /// game.code
var mapBetsByGameCode sync.Map // by lobby id

// func init() {
// 	mapGameByCode = sync.Map{}
// }

func InitListGame(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule) {
	gamesJson := `{"games":[{"code":"color-game","layout":{"col":1,"row":2,"col_span":2,"row_span":2},"id":2},{"code":"roulette","layout":{"col":1,"row":3,"col_span":2,"row_span":2},"id":4},{"code":"fruit-slot","layout":{"col":1,"row":4,"col_span":2,"row_span":2},"id":5},{"code":"sabong-cards","layout":{"col":2,"row":1,"col_span":2,"row_span":2},"id":6},{"code":"chinese-poker","layout":{"col":2,"row":2,"col_span":2,"row_span":2},"id":7},{"code":"baccarat","layout":{"col":2,"row":3,"col_span":2,"row_span":2},"id":8},{"code":"lucky-number","layout":{"col":2,"row":4,"col_span":2,"row_span":2},"id":9},{"code":"sixiang","layout":{"col":2,"row":4,"col_span":2,"row_span":2},"id":13},{"code":"tarzan","layout":{"col":2,"row":4,"col_span":2,"row_span":2},"id":12},{"code":"juicygarden","layout":{"col":2,"row":4,"col_span":2,"row_span":2},"id":11},{"code":"blackjack","layout":{"col":2,"row":4,"col_span":2,"row_span":2},"id":14},{"code":"bandarqq","layout":{"col":2,"row":4,"col_span":2,"row_span":2},"id":15},{"code":"sicbo","layout":{"col":2,"row":4,"col_span":2,"row_span":2},"id":16},{"code":"dragontiger","layout":{"col":2,"row":4,"col_span":2,"row_span":2},"id":17},{"code":"inca","layout":{"col":2,"row":4,"col_span":2,"row_span":2},"id":18},{"code":"noel","layout":{"col":2,"row":4,"col_span":2,"row_span":2},"id":19},{"code":"fruit","layout":{"col":2,"row":4,"col_span":2,"row_span":2},"id":20},{"code":"gaple","layout":{"col":2,"row":4,"col_span":2,"row_span":2},"id":21}]}`
	games := entity.Games{}
	json.Unmarshal([]byte(gamesJson), &games)
	for _, game := range games.List {
		err := cgbdb.AddGame(ctx, db, &game)
		if err != nil {
			logger.WithField("err", err).Error("add games failed")
		}
	}

	cacheListGame(ctx, db, logger)

}

func RpcGameAdd(marshaler *protojson.MarshalOptions, unmarshaler *protojson.UnmarshalOptions) func(context.Context, runtime.Logger, *sql.DB, runtime.NakamaModule, string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		userID, _ := ctx.Value(runtime.RUNTIME_CTX_USER_ID).(string)
		if userID != "" {
			return "", errors.New("Unauth")
		}
		game := &entity.Game{}
		if err := json.Unmarshal([]byte(payload), game); err != nil {
			logger.Error("Error when unmarshal payload", err.Error())
			return "", presenter.ErrUnmarshal
		}
		err := cgbdb.AddGame(ctx, db, game)
		if err == nil {
			cacheListGame(ctx, db, logger)
		}
		return "", err
	}
}

func RpcGameList(marshaler *protojson.MarshalOptions, unmarshaler *protojson.UnmarshalOptions) func(context.Context, runtime.Logger, *sql.DB, runtime.NakamaModule, string) (string, error) {
	return func(ctx context.Context, logger runtime.Logger, db *sql.DB, nk runtime.NakamaModule, payload string) (string, error) {
		ml, err := cgbdb.ListGames(ctx, db)
		if err != nil {
			return "", err
		}
		if len(ml) == 0 {
			return "", nil
		}
		listGameCode := make([]string, 0)
		for _, game := range ml {
			listGameCode = append(listGameCode, game.Code)
		}
		jackpots, err := cgbdb.GetJackpotsByGame(ctx, logger, db, listGameCode...)
		if err == nil {
			mapJp := make(map[string]*pb.Jackpot)
			for _, jp := range jackpots {
				v := jp
				if v.GameCode == define.ChinesePoker.String() {
					if v.Chips < define.MinJpTreasure {
						v.Chips = define.MinJpTreasure
						cgbdb.SetJackpot(ctx, logger, db, v.GameCode, v.Chips)
					}
				}
				mapJp[v.GameCode] = v
			}
			for idx, game := range ml {
				if mapJp[game.Code] == nil {
					continue
				}
				game.JpChips = mapJp[game.Code].Chips
				ml[idx] = game
			}
		}
		games := &entity.Games{
			List: ml,
		}
		dataJson, _ := json.Marshal(games)
		return string(dataJson), nil
	}
}

func cacheListGame(ctx context.Context, db *sql.DB, logger runtime.Logger) {
	ml, err := cgbdb.ListGames(ctx, db)
	if err != nil {
		logger.WithField("err", err).Error("load list game failed")
		return
	}
	for _, game := range ml {
		v := game
		mapGameByCode.Store(game.Code, v)
	}

}
