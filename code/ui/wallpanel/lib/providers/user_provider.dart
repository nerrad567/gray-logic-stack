import 'package:flutter_riverpod/flutter_riverpod.dart';

import '../models/user.dart';
import '../repositories/user_repository.dart';
import 'auth_provider.dart';

/// Provides the UserRepository.
final userRepositoryProvider = Provider<UserRepository>((ref) {
  return UserRepository(apiClient: ref.watch(apiClientProvider));
});

/// Provides all users (admin view).
final allUsersProvider =
    NotifierProvider<AllUsersNotifier, AsyncValue<List<User>>>(
  AllUsersNotifier.new,
);

class AllUsersNotifier extends Notifier<AsyncValue<List<User>>> {
  @override
  AsyncValue<List<User>> build() => const AsyncValue.loading();

  UserRepository get _userRepo => ref.read(userRepositoryProvider);

  /// Load all users.
  Future<void> load() async {
    state = const AsyncValue.loading();
    try {
      final users = await _userRepo.getUsers();
      users.sort((a, b) => a.username.compareTo(b.username));
      state = AsyncValue.data(users);
    } catch (e, st) {
      state = AsyncValue.error(e, st);
    }
  }

  /// Create a new user and refresh list.
  Future<User> createUser(Map<String, dynamic> data) async {
    final user = await _userRepo.createUser(data);
    await load();
    return user;
  }

  /// Update a user and refresh list.
  Future<User> updateUser(String id, Map<String, dynamic> data) async {
    final user = await _userRepo.updateUser(id, data);
    await load();
    return user;
  }

  /// Delete a user and refresh list.
  Future<void> deleteUser(String id) async {
    await _userRepo.deleteUser(id);
    await load();
  }
}
