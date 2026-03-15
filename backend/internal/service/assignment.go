package service

import (
	"context"
	"math/rand"
	"sort"
	"time"

	"github.com/google/uuid"

	"comp-video-service/backend/internal/model"
	"comp-video-service/backend/internal/repository"
)

// AssignmentService assigns randomized tasks to participants.
type AssignmentService struct {
	sourceItemRepo *repository.SourceItemRepository
	groupRepo      *repository.GroupRepository
	videoRepo      *repository.VideoRepository
	pairRepo       *repository.PairPresentationRepository
}

func NewAssignmentService(
	sourceItemRepo *repository.SourceItemRepository,
	groupRepo *repository.GroupRepository,
	videoRepo *repository.VideoRepository,
	pairRepo *repository.PairPresentationRepository,
) *AssignmentService {
	return &AssignmentService{
		sourceItemRepo: sourceItemRepo,
		groupRepo:      groupRepo,
		videoRepo:      videoRepo,
		pairRepo:       pairRepo,
	}
}

type assignmentCandidate struct {
	item          *model.SourceItem
	assets        []*model.Video
	groupPriority int
	deficit       int64
	randTie       int64
}

// AssignForParticipant creates balanced randomized tasks for the participant.
func (s *AssignmentService) AssignForParticipant(ctx context.Context, participantID, studyID uuid.UUID, maxTasks int) (int, error) {
	items, err := s.sourceItemRepo.ListByStudy(ctx, studyID)
	if err != nil {
		return 0, err
	}
	if len(items) == 0 {
		return 0, nil
	}

	groups, err := s.groupRepo.ListByStudy(ctx, studyID)
	if err != nil {
		return 0, err
	}
	responseCounts, err := s.sourceItemRepo.ResponseCountsByStudy(ctx, studyID)
	if err != nil {
		return 0, err
	}

	groupTarget := make(map[uuid.UUID]int64, len(groups))
	groupPriority := make(map[uuid.UUID]int, len(groups))
	for _, g := range groups {
		target := int64(g.TargetVotesPerPair)
		if target <= 0 {
			target = 10
		}
		groupTarget[g.ID] = target
		groupPriority[g.ID] = g.Priority
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	underTargetByGroup := make(map[uuid.UUID][]*assignmentCandidate)
	groupTotalDeficit := make(map[uuid.UUID]int64)
	metTarget := make([]*assignmentCandidate, 0)

	for _, item := range items {
		assets, err := s.videoRepo.ListBySourceItem(ctx, item.ID)
		if err != nil {
			return 0, err
		}
		if len(assets) < 2 {
			continue
		}

		target := groupTarget[item.GroupID]
		if target <= 0 {
			target = 10
		}
		current := responseCounts[item.ID]
		deficit := target - current
		if deficit < 0 {
			deficit = 0
		}

		candidate := &assignmentCandidate{
			item:          item,
			assets:        assets,
			groupPriority: groupPriority[item.GroupID],
			deficit:       deficit,
			randTie:       rng.Int63(),
		}

		if deficit > 0 {
			underTargetByGroup[item.GroupID] = append(underTargetByGroup[item.GroupID], candidate)
			groupTotalDeficit[item.GroupID] += deficit
		} else {
			metTarget = append(metTarget, candidate)
		}
	}

	totalCandidates := len(metTarget)
	for _, cands := range underTargetByGroup {
		totalCandidates += len(cands)
	}
	if totalCandidates == 0 {
		return 0, nil
	}

	if maxTasks <= 0 || maxTasks > totalCandidates {
		maxTasks = totalCandidates
	}

	for gid := range underTargetByGroup {
		sort.Slice(underTargetByGroup[gid], func(i, j int) bool {
			if underTargetByGroup[gid][i].deficit != underTargetByGroup[gid][j].deficit {
				return underTargetByGroup[gid][i].deficit > underTargetByGroup[gid][j].deficit
			}
			return underTargetByGroup[gid][i].randTie < underTargetByGroup[gid][j].randTie
		})
	}

	groupOrder := make([]uuid.UUID, 0, len(underTargetByGroup))
	for gid := range underTargetByGroup {
		groupOrder = append(groupOrder, gid)
	}
	sort.Slice(groupOrder, func(i, j int) bool {
		pi := groupPriority[groupOrder[i]]
		pj := groupPriority[groupOrder[j]]
		if pi != pj {
			return pi > pj
		}
		di := groupTotalDeficit[groupOrder[i]]
		dj := groupTotalDeficit[groupOrder[j]]
		if di != dj {
			return di > dj
		}
		return groupOrder[i].String() < groupOrder[j].String()
	})

	selected := make([]*assignmentCandidate, 0, maxTasks)
	for len(selected) < maxTasks {
		picked := false
		for _, gid := range groupOrder {
			queue := underTargetByGroup[gid]
			if len(queue) == 0 {
				continue
			}
			selected = append(selected, queue[0])
			underTargetByGroup[gid] = queue[1:]
			picked = true
			if len(selected) >= maxTasks {
				break
			}
		}
		if !picked {
			break
		}
	}

	if len(selected) < maxTasks {
		rng.Shuffle(len(metTarget), func(i, j int) {
			metTarget[i], metTarget[j] = metTarget[j], metTarget[i]
		})
		remaining := maxTasks - len(selected)
		if remaining > len(metTarget) {
			remaining = len(metTarget)
		}
		selected = append(selected, metTarget[:remaining]...)
	}

	created := 0
	for idx, candidate := range selected {
		left := candidate.assets[0]
		right := candidate.assets[1]
		if rng.Intn(2) == 1 {
			left, right = right, left
		}

		_, err := s.pairRepo.Create(ctx, &model.PairPresentation{
			ParticipantID:    participantID,
			SourceItemID:     candidate.item.ID,
			LeftAssetID:      left.ID,
			RightAssetID:     right.ID,
			LeftMethodType:   derefOrEmpty(left.MethodType),
			RightMethodType:  derefOrEmpty(right.MethodType),
			TaskOrder:        idx + 1,
			IsAttentionCheck: candidate.item.IsAttentionCheck,
			IsPractice:       false,
		})
		if err != nil {
			return created, err
		}
		created++
	}

	return created, nil
}

func derefOrEmpty(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}
