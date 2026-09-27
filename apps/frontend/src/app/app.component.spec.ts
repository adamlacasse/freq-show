import { Component, signal } from '@angular/core';
import { TestBed } from '@angular/core/testing';
import { provideRouter, Router } from '@angular/router';
import { of } from 'rxjs';
import { AppComponent } from './app.component';
import { SearchService } from './services/search.service';
import { ApiStatusService } from './services/api-status.service';
import { AuthService, User } from './services/auth.service';

@Component({
  standalone: true,
  template: ''
})
class DummyComponent {}

describe('AppComponent', () => {
  let searchServiceSpy: jasmine.SpyObj<SearchService>;
  let apiStatusServiceStub: Pick<ApiStatusService, 'init' | 'status$'>;
  let currentUserSignal: ReturnType<typeof signal<User | null>>;
  let isAuthenticatedSignal: ReturnType<typeof signal<boolean>>;
  let authServiceStub: any;

  beforeEach(async () => {
    searchServiceSpy = jasmine.createSpyObj<SearchService>('SearchService', ['requestSearchReset']);
    apiStatusServiceStub = {
      init: jasmine.createSpy('init'),
      status$: of('ready')
    };

    currentUserSignal = signal<User | null>(null);
    isAuthenticatedSignal = signal<boolean>(false);
    authServiceStub = {
      currentUser: currentUserSignal,
      isAuthenticated: isAuthenticatedSignal,
      isModalOpen: signal<boolean>(false),
      checkSession: jasmine.createSpy('checkSession').and.returnValue(of(null)),
      openModal: jasmine.createSpy('openModal'),
      closeModal: jasmine.createSpy('closeModal'),
      logout: jasmine.createSpy('logout').and.returnValue(of({ status: 'ok' }))
    };

    await TestBed.configureTestingModule({
      imports: [AppComponent],
      providers: [
        provideRouter([
          { path: '', component: DummyComponent },
          { path: 'artists/:id', component: DummyComponent },
          { path: 'collections/:userId', component: DummyComponent },
        ]),
        { provide: SearchService, useValue: searchServiceSpy },
        { provide: ApiStatusService, useValue: apiStatusServiceStub },
        { provide: AuthService, useValue: authServiceStub }
      ],
    }).compileComponents();
  });

  it('should create the app', () => {
    const fixture = TestBed.createComponent(AppComponent);
    const app = fixture.componentInstance;
    expect(app).toBeTruthy();
  });

  it(`should have the 'FreqShow!' title`, () => {
    const fixture = TestBed.createComponent(AppComponent);
    const app = fixture.componentInstance;
    expect(app.title).toEqual('FreqShow!');
  });

  it('should render the brand in the header', () => {
    const fixture = TestBed.createComponent(AppComponent);
    fixture.detectChanges();
    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.querySelector('header a')?.textContent).toContain('FreqShow!');
  });

  it('should clear search state when the home link is used', () => {
    const fixture = TestBed.createComponent(AppComponent);
    const app = fixture.componentInstance;

    app.onHomeClick();

    expect(searchServiceSpy.requestSearchReset).toHaveBeenCalled();
  });

  it('should clear search state after navigating away from home', async () => {
    const fixture = TestBed.createComponent(AppComponent);
    fixture.detectChanges();

    const router = TestBed.inject(Router);
    await router.navigateByUrl('/artists/123');
    await fixture.whenStable();

    expect(searchServiceSpy.requestSearchReset).toHaveBeenCalled();
  });

  it('should render Sign In button when unauthenticated and open modal on click', () => {
    const fixture = TestBed.createComponent(AppComponent);
    fixture.detectChanges();

    const compiled = fixture.nativeElement as HTMLElement;
    const signInBtn = Array.from(compiled.querySelectorAll('button')).find(
      (b) => b.textContent?.trim().includes('Sign In')
    );
    expect(signInBtn).toBeTruthy();

    signInBtn?.click();
    expect(authServiceStub.openModal).toHaveBeenCalled();
  });

  it('should render user email and Sign Out button when authenticated', () => {
    currentUserSignal.set({
      id: 'usr-42',
      email: 'collector@example.com',
      createdAt: '2026-09-27T00:00:00Z'
    });
    isAuthenticatedSignal.set(true);

    const fixture = TestBed.createComponent(AppComponent);
    fixture.detectChanges();

    const compiled = fixture.nativeElement as HTMLElement;
    expect(compiled.textContent).toContain('collector@example.com');

    const signOutBtn = Array.from(compiled.querySelectorAll('button')).find(
      (b) => b.textContent?.trim().includes('Sign Out')
    );
    expect(signOutBtn).toBeTruthy();

    signOutBtn?.click();
    expect(authServiceStub.logout).toHaveBeenCalled();
  });

  it('should dynamically link to user collection when authenticated', () => {
    currentUserSignal.set({
      id: 'usr-99',
      email: 'user99@example.com',
      createdAt: '2026-09-27T00:00:00Z'
    });
    isAuthenticatedSignal.set(true);

    const fixture = TestBed.createComponent(AppComponent);
    fixture.detectChanges();

    expect(fixture.componentInstance.collectionRoute()).toBe('/collections/usr-99');
  });
});
