import 'package:flutter/foundation.dart';

import '../../../../core/network/api_client.dart';
import '../../../../core/network/api_exception.dart';

class HomeProfile {
  const HomeProfile({
    required this.id,
    required this.email,
    required this.name,
  });

  final int id;
  final String email;
  final String name;
}

class HomePreferences {
  const HomePreferences({
    required this.reminderIntensity,
    required this.pushEnabled,
    required this.emailEnabled,
    required this.whatsappEnabled,
  });

  final String reminderIntensity;
  final bool pushEnabled;
  final bool emailEnabled;
  final bool whatsappEnabled;
}

class HomeEvent {
  const HomeEvent({
    required this.id,
    required this.title,
    required this.description,
    required this.startAt,
    required this.endAt,
    required this.isAllDay,
    required this.isCompleted,
  });

  final int id;
  final String title;
  final String description;
  final DateTime startAt;
  final DateTime endAt;
  final bool isAllDay;
  final bool isCompleted;
}

class EventDraft {
  const EventDraft({
    required this.title,
    required this.description,
    required this.startAt,
    required this.endAt,
    this.timezone = 'UTC',
    this.isAllDay = false,
  });

  final String title;
  final String description;
  final DateTime startAt;
  final DateTime endAt;
  final String timezone;
  final bool isAllDay;
}

class HomeGamificationSummary {
  const HomeGamificationSummary({
    required this.xp,
    required this.level,
    required this.currentStreak,
    required this.longestStreak,
    this.lastActivityDay,
  });

  final int xp;
  final int level;
  final int currentStreak;
  final int longestStreak;
  final String? lastActivityDay;
}

class HomeAchievement {
  const HomeAchievement({
    required this.key,
    required this.title,
    required this.description,
    this.unlockedAt,
  });

  final String key;
  final String title;
  final String description;
  final DateTime? unlockedAt;
}

class HomeController extends ChangeNotifier {
  HomeController({required this.apiClient});

  final ApiClient apiClient;

  bool _isLoading = false;
  String? _errorMessage;
  HomeProfile? _profile;
  HomePreferences _preferences = const HomePreferences(
    reminderIntensity: 'medium',
    pushEnabled: true,
    emailEnabled: true,
    whatsappEnabled: false,
  );
  HomeGamificationSummary? _summary;
  List<HomeEvent> _events = const [];
  List<HomeAchievement> _achievements = const [];

  bool get isLoading => _isLoading;
  String? get errorMessage => _errorMessage;
  HomeProfile? get profile => _profile;
  HomePreferences get preferences => _preferences;
  HomeGamificationSummary? get summary => _summary;
  List<HomeEvent> get events => _events;
  List<HomeAchievement> get achievements => _achievements;

  Future<void> updateProfile(String name) async {
    final trimmedName = name.trim();
    final response = await apiClient.put(
      '/api/v1/profile',
      requiresAuth: true,
      body: {'name': trimmedName},
    );

    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw ApiException('Falha ao atualizar perfil (${response.statusCode}).', statusCode: response.statusCode);
    }

    final currentProfile = _profile;
    if (currentProfile != null) {
      _profile = HomeProfile(id: currentProfile.id, email: currentProfile.email, name: trimmedName);
      notifyListeners();
    } else {
      await load();
    }
  }

  Future<void> updatePreferences(HomePreferences preferences) async {
    final response = await apiClient.put(
      '/api/v1/preferences',
      requiresAuth: true,
      body: {
        'reminder_intensity': preferences.reminderIntensity,
        'push_enabled': preferences.pushEnabled,
        'email_enabled': preferences.emailEnabled,
        'whatsapp_enabled': preferences.whatsappEnabled,
      },
    );

    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw ApiException('Falha ao atualizar preferencias (${response.statusCode}).', statusCode: response.statusCode);
    }

    final data = apiClient.decodeJsonObject(response.body);
    _preferences = HomePreferences(
      reminderIntensity: data['reminder_intensity']?.toString() ?? preferences.reminderIntensity,
      pushEnabled: data['push_enabled'] == true,
      emailEnabled: data['email_enabled'] == true,
      whatsappEnabled: data['whatsapp_enabled'] == true,
    );
    notifyListeners();
  }

  Future<void> createEvent(EventDraft draft) async {
    await _mutateEvent(() {
      return apiClient.post(
        '/api/v1/events',
        requiresAuth: true,
        body: _eventPayload(draft),
      );
    });
  }

  Future<void> updateEvent(int eventId, EventDraft draft) async {
    await _mutateEvent(() {
      return apiClient.put(
        '/api/v1/events/$eventId',
        requiresAuth: true,
        body: _eventPayload(draft),
      );
    });
  }

  Future<void> deleteEvent(int eventId) async {
    await _mutateEvent(() {
      return apiClient.delete('/api/v1/events/$eventId', requiresAuth: true);
    }, allowNoContent: true);
  }

  Future<void> completeEvent(int eventId) async {
    await _mutateEvent(() {
      return apiClient.post('/api/v1/events/$eventId/complete', requiresAuth: true);
    });
  }

  Future<void> load() async {
    _isLoading = true;
    _errorMessage = null;
    notifyListeners();

    try {
      final now = DateTime.now().toUtc();
      final from = now.subtract(const Duration(days: 1)).toIso8601String();
      final to = now.add(const Duration(days: 14)).toIso8601String();

      final responses = await Future.wait([
        apiClient.get('/api/v1/profile', requiresAuth: true),
        apiClient.get('/api/v1/gamification/summary', requiresAuth: true),
        apiClient.get('/api/v1/gamification/achievements', requiresAuth: true),
        apiClient.get(
          '/api/v1/events?page=1&limit=5&from=${Uri.encodeQueryComponent(from)}&to=${Uri.encodeQueryComponent(to)}',
          requiresAuth: true,
        ),
      ]);

      for (final response in responses) {
        if (response.statusCode < 200 || response.statusCode >= 300) {
          throw ApiException('Falha ao carregar painel (${response.statusCode}).', statusCode: response.statusCode);
        }
      }

      final profileData = apiClient.decodeJsonObject(responses[0].body);
      _profile = HomeProfile(
        id: (profileData['id'] as num?)?.toInt() ?? 0,
        email: profileData['email']?.toString() ?? '',
        name: profileData['name']?.toString() ?? '',
      );

      final summaryData = apiClient.decodeJsonObject(responses[1].body);
      _summary = HomeGamificationSummary(
        xp: (summaryData['xp'] as num?)?.toInt() ?? 0,
        level: (summaryData['level'] as num?)?.toInt() ?? 1,
        currentStreak: (summaryData['current_streak'] as num?)?.toInt() ?? 0,
        longestStreak: (summaryData['longest_streak'] as num?)?.toInt() ?? 0,
        lastActivityDay: summaryData['last_activity_day']?.toString(),
      );

      final achievementsData = apiClient.decodeJsonObject(responses[2].body);
      final achievementsItems = achievementsData['achievements'] as List<dynamic>? ?? const [];
      _achievements = achievementsItems
          .whereType<Map>()
          .map(
            (item) => HomeAchievement(
              key: item['key']?.toString() ?? '',
              title: item['title']?.toString() ?? '',
              description: item['description']?.toString() ?? '',
              unlockedAt: DateTime.tryParse(item['unlocked_at']?.toString() ?? ''),
            ),
          )
          .toList();

      final eventsData = apiClient.decodeJsonObject(responses[3].body);
      final eventItems = eventsData['events'] as List<dynamic>? ?? const [];
      _events = eventItems.whereType<Map>().map(_mapHomeEvent).toList();

      await _loadPreferences();
    } catch (error) {
      _errorMessage = error.toString();
    } finally {
      _isLoading = false;
      notifyListeners();
    }
  }

  Map<String, dynamic> _eventPayload(EventDraft draft) {
    return {
      'title': draft.title.trim(),
      'description': draft.description.trim(),
      'start_at': draft.startAt.toUtc().toIso8601String(),
      'end_at': draft.endAt.toUtc().toIso8601String(),
      'timezone': draft.timezone,
      'is_all_day': draft.isAllDay,
      'recurrence': const <String, dynamic>{},
      'reminder_offsets_minutes': const [60, 15],
    };
  }

  HomeEvent _mapHomeEvent(Map item) {
    final fallback = DateTime.now().toUtc();
    return HomeEvent(
      id: (item['id'] as num?)?.toInt() ?? 0,
      title: item['title']?.toString() ?? '',
      description: item['description']?.toString() ?? '',
      startAt: DateTime.tryParse(item['start_at']?.toString() ?? '') ?? fallback,
      endAt: DateTime.tryParse(item['end_at']?.toString() ?? '') ?? fallback,
      isAllDay: item['is_all_day'] == true,
      isCompleted: item['completed_at'] != null,
    );
  }

  Future<void> _loadPreferences() async {
    try {
      final response = await apiClient.get('/api/v1/preferences', requiresAuth: true);
      if (response.statusCode == 404) {
        _preferences = const HomePreferences(
          reminderIntensity: 'medium',
          pushEnabled: true,
          emailEnabled: true,
          whatsappEnabled: false,
        );
        return;
      }
      if (response.statusCode < 200 || response.statusCode >= 300) {
        _preferences = const HomePreferences(
          reminderIntensity: 'medium',
          pushEnabled: true,
          emailEnabled: true,
          whatsappEnabled: false,
        );
        return;
      }

      final preferencesData = apiClient.decodeJsonObject(response.body);
      _preferences = HomePreferences(
        reminderIntensity: preferencesData['reminder_intensity']?.toString() ?? 'medium',
        pushEnabled: preferencesData['push_enabled'] == true,
        emailEnabled: preferencesData['email_enabled'] == true,
        whatsappEnabled: preferencesData['whatsapp_enabled'] == true,
      );
    } catch (_) {
      _preferences = const HomePreferences(
        reminderIntensity: 'medium',
        pushEnabled: true,
        emailEnabled: true,
        whatsappEnabled: false,
      );
    }
  }

  Future<void> _mutateEvent(
    Future<dynamic> Function() request, {
    bool allowNoContent = false,
  }) async {
    final response = await request();
    if (!allowNoContent && (response.statusCode < 200 || response.statusCode >= 300)) {
      throw ApiException('Falha ao salvar evento (${response.statusCode}).', statusCode: response.statusCode);
    }
    if (allowNoContent && response.statusCode != 204 && (response.statusCode < 200 || response.statusCode >= 300)) {
      throw ApiException('Falha ao atualizar evento (${response.statusCode}).', statusCode: response.statusCode);
    }
    await load();
  }
}
