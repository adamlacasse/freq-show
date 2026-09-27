import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { HttpTestingController, provideHttpClientTesting } from '@angular/common/http/testing';
import { AuthService, User, AuthRequestResponse, AuthLogoutResponse } from './auth.service';

describe('AuthService', () => {
  let service: AuthService;
  let httpTesting: HttpTestingController;

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        AuthService
      ]
    });

    service = TestBed.inject(AuthService);
    httpTesting = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpTesting.verify();
  });

  it('should initialize with no user and unauthenticated', () => {
    expect(service.currentUser()).toBeNull();
    expect(service.isAuthenticated()).toBeFalse();
    expect(service.isModalOpen()).toBeFalse();
  });

  describe('checkSession', () => {
    it('should set currentUser when user is authenticated', () => {
      const mockUser: User = {
        id: 'usr-1',
        email: 'listener@example.com',
        createdAt: '2026-09-27T00:00:00Z'
      };

      service.checkSession().subscribe((user) => {
        expect(user).toEqual(mockUser);
      });

      const req = httpTesting.expectOne('/api/auth/me');
      expect(req.request.method).toBe('GET');
      req.flush(mockUser);

      expect(service.currentUser()).toEqual(mockUser);
      expect(service.isAuthenticated()).toBeTrue();
      expect(service.isCheckingSession()).toBeFalse();
    });

    it('should reset currentUser to null when unauthenticated (401)', () => {
      service.checkSession().subscribe((user) => {
        expect(user).toBeNull();
      });

      const req = httpTesting.expectOne('/api/auth/me');
      expect(req.request.method).toBe('GET');
      req.flush({ error: 'not authenticated' }, { status: 401, statusText: 'Unauthorized' });

      expect(service.currentUser()).toBeNull();
      expect(service.isAuthenticated()).toBeFalse();
      expect(service.isCheckingSession()).toBeFalse();
    });
  });

  describe('requestMagicLink', () => {
    it('should post email to /api/auth/request', () => {
      const mockResponse: AuthRequestResponse = {
        status: 'ok',
        message: 'check your email for a sign-in link'
      };

      service.requestMagicLink('test@example.com').subscribe((res) => {
        expect(res).toEqual(mockResponse);
      });

      const req = httpTesting.expectOne('/api/auth/request');
      expect(req.request.method).toBe('POST');
      expect(req.request.body).toEqual({ email: 'test@example.com' });
      req.flush(mockResponse);
    });
  });

  describe('logout', () => {
    it('should post to /api/auth/logout and clear user', () => {
      service.currentUser.set({
        id: 'usr-1',
        email: 'listener@example.com',
        createdAt: '2026-09-27T00:00:00Z'
      });
      expect(service.isAuthenticated()).toBeTrue();

      const mockResponse: AuthLogoutResponse = {
        status: 'ok',
        message: 'signed out'
      };

      service.logout().subscribe((res) => {
        expect(res.status).toBe('ok');
      });

      const req = httpTesting.expectOne('/api/auth/logout');
      expect(req.request.method).toBe('POST');
      req.flush(mockResponse);

      expect(service.currentUser()).toBeNull();
      expect(service.isAuthenticated()).toBeFalse();
    });
  });

  describe('modal visibility', () => {
    it('should toggle modal visibility signals', () => {
      expect(service.isModalOpen()).toBeFalse();

      service.openModal();
      expect(service.isModalOpen()).toBeTrue();

      service.closeModal();
      expect(service.isModalOpen()).toBeFalse();
    });
  });
});
