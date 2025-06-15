package services

import (
	"fmt"
	"sync"
	"time"

	"github.com/bradselph/CODStatusBot/configuration"
	"github.com/bradselph/CODStatusBot/logger"
	"github.com/bwmarrin/discordgo"
)

type ShardManager struct {
	sync.Mutex
	Sessions       []*discordgo.Session
	MaxConcurrency int
	TotalShards    int
	StartedShards  int
}

func NewShardManager() (*ShardManager, error) {
	cfg := configuration.Get()

	gatewayInfo, err := GetGatewayBotInfo(cfg.Discord.Token)
	if err != nil {
		return nil, fmt.Errorf("failed to get gateway bot info: %w", err)
	}

	logger.Log.Infof("Creating shard manager with %d shards and max concurrency %d",
		gatewayInfo.Shards, gatewayInfo.SessionStartLimit.MaxConcurrency)

	return &ShardManager{
		Sessions:       make([]*discordgo.Session, gatewayInfo.Shards),
		MaxConcurrency: gatewayInfo.SessionStartLimit.MaxConcurrency,
		TotalShards:    gatewayInfo.Shards,
		StartedShards:  0,
	}, nil
}

func GetGatewayBotInfo(token string) (*discordgo.GatewayBotResponse, error) {
	s, err := discordgo.New("Bot " + token)
	if err != nil {
		return nil, err
	}

	return s.GatewayBot()
}

func (sm *ShardManager) StartShards(token string, handlers map[string]func(*discordgo.Session, *discordgo.InteractionCreate)) error {
	sm.Lock()
	defer sm.Unlock()

	buckets := make(map[int]bool)

	for i := 0; i < sm.TotalShards; i++ {
		bucket := i % sm.MaxConcurrency

		if i >= sm.MaxConcurrency && !buckets[bucket] {
			logger.Log.Infof("Waiting for previous bucket to complete before starting shard %d (bucket %d)", i, bucket)
			time.Sleep(5 * time.Second)
		}

		buckets[bucket] = true

		logger.Log.Infof("Starting shard %d of %d", i, sm.TotalShards)
		session, err := discordgo.New("Bot " + token)
		if err != nil {
			return fmt.Errorf("error creating session for shard %d: %w", i, err)
		}

		session.ShardID = i
		session.ShardCount = sm.TotalShards

		registerHandlers(session, handlers)

		err = session.Open()
		if err != nil {
			return fmt.Errorf("error opening session for shard %d: %w", i, err)
		}

		sm.Sessions[i] = session
		sm.StartedShards++

		logger.Log.Infof("Shard %d started successfully", i)

		if sm.MaxConcurrency > 1 {
			time.Sleep(1 * time.Second)
		}
	}

	logger.Log.Infof("All %d shards started successfully", sm.TotalShards)
	return nil
}

func (sm *ShardManager) Close() {
	sm.Lock()
	defer sm.Unlock()

	for i, s := range sm.Sessions {
		if s != nil {
			logger.Log.Infof("Closing shard %d", i)
			s.Close()
		}
	}
}

func (sm *ShardManager) GetSession(guildID string) *discordgo.Session {
	if len(sm.Sessions) == 1 {
		return sm.Sessions[0]
	}

	shardID := getShardIDForGuild(guildID, sm.TotalShards)

	if shardID >= 0 && shardID < len(sm.Sessions) {
		return sm.Sessions[shardID]
	}

	return sm.Sessions[0]
}

func registerHandlers(s *discordgo.Session, handlers map[string]func(*discordgo.Session, *discordgo.InteractionCreate)) {
	s.AddHandler(func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		switch i.Type {
		case discordgo.InteractionApplicationCommand:
			if h, ok := handlers[i.ApplicationCommandData().Name]; ok {
				h(s, i)
			}
		case discordgo.InteractionModalSubmit:
			if h, ok := handlers[i.ModalSubmitData().CustomID]; ok {
				h(s, i)
			}
		case discordgo.InteractionMessageComponent:
			if h, ok := handlers[i.MessageComponentData().CustomID]; ok {
				h(s, i)
			}
		}
	})
}

func getShardIDForGuild(guildID string, shardCount int) int {
	if guildID == "" {
		return 0
	}

	var id uint64
	fmt.Sscanf(guildID, "%d", &id)

	return int((id >> 22) % uint64(shardCount))
}
